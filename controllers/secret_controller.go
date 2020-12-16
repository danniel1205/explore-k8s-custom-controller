/*

 */

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pod", req.NamespacedName)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		log.Error(err, "unable to get pod")
		return ctrl.Result{}, err
	}

	pod.ObjectMeta.Annotations["reconciled-from-secret"] = "true"
	if err := r.Update(ctx, pod); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// enqueue the pod for reconcile
	mapFn := func(o client.Object) []reconcile.Request {
		if o.GetAnnotations()["pod-name"] != "" {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      o.GetAnnotations()["pod-name"],
					Namespace: o.GetNamespace(),
				}},
			}
		} else {
			return []reconcile.Request{}
		}
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// The object doesn't contain label "foo", so the event will be
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
	}

	fmt.Println("==> SetupWithManager")

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(mapFn),
			builder.OnlyMetadata).
		WithEventFilter(p).
		Complete(r)

	//return ctrl.NewControllerManagedBy(mgr).
	//	For(&corev1.Secret{}, builder.OnlyMetadata).
	//	Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
