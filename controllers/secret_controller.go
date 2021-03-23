/*

 */

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets;pods;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status;pods/status;configmaps/status,verbs=get;update;patch

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	log := r.Log.WithValues("pod", req.NamespacedName)

	pod := &corev1.Pod{}
	if err = r.Get(ctx, req.NamespacedName, pod); err != nil {
		if errors.IsNotFound(err) {
			log.Info("create pod", "namespace", req.Namespace, "name", req.Name)
			// create pod
			if err = r.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox",
							Args: []string{
								"sleep",
								"1000000",
							},
						},
					},
				},
			}); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// enqueue the request to reconcile Pod
	enqueueMapFn := func(o client.Object) []reconcile.Request {
		podName := o.GetAnnotations()["pod-name"]
		podNamespace := o.GetNamespace()
		if podName != "" {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      podName,
					Namespace: podNamespace,
				}},
			}
		} else {
			return []reconcile.Request{}
		}
	}

	eventFilter := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// If the object doesn't contain annotation "pod-name", so the event will be
			// ignored.
			if _, ok := e.ObjectNew.GetAnnotations()["pod-name"]; !ok {
				return false
			}
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			if _, ok := e.Object.GetAnnotations()["pod-name"]; !ok {
				return false
			}
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if _, ok := e.Object.GetAnnotations()["pod-name"]; !ok {
				return false
			}
			return true
		},
	}

	// Watch on the configMap and reconcile Pods
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(enqueueMapFn),
			builder.OnlyMetadata).
		WithEventFilter(eventFilter).
		Complete(r)
}
