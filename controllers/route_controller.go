/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RouteReconciler reconciles a Route object
type RouteReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Route object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *RouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("route", req.NamespacedName)

	// Lookup the route instance for this reconcile request
	route := &routev1.Route{}
	err := r.Get(ctx, req.NamespacedName, route)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Routes")
		return ctrl.Result{}, err
	}

	for _, ref := range route.OwnerReferences {
		if ref.Kind == "Ingress" {
			log.Info("Ignoring route as it is owned by ingress", "Ingress.Namespace", route.Namespace, "Ingress.Name", route.Name)
			return ctrl.Result{}, err
		}
	}

	// Check if the ingress already exists, if not create a new one
	found := &netv1.Ingress{}
	err = r.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ingress
		ing, err := r.ingressForRoute(route)
		if err != nil {
			log.Error(err, "Error converting route to ing")
			return ctrl.Result{}, nil
		}
		log.Info("Creating a new ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
		err = r.Create(ctx, ing)
		if err != nil {
			log.Error(err, "Failed to create new ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Ingress")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RouteReconciler) ingressForRoute(m *routev1.Route) (*netv1.Ingress, error) {
	if m.Spec.Port == nil {
		return nil, fmt.Errorf("nil port")
	}
	var pathType netv1.PathType = "Exact"
	var port netv1.ServiceBackendPort
	switch m.Spec.Port.TargetPort.Type {
	case intstr.String:
		port = netv1.ServiceBackendPort{
			Name: m.Spec.Port.TargetPort.StrVal,
		}
	case intstr.Int:
		port = netv1.ServiceBackendPort{
			Number: m.Spec.Port.TargetPort.IntVal,
		}
	default:
		return nil, fmt.Errorf("unknown targetport")
	}

	var path string
	if m.Spec.Path == "" {
		path = "/"
	} else {
		path = m.Spec.Path
	}
	ing := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    m.Labels,
		},

		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: m.Spec.Host,
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path:     path,
							PathType: &pathType,
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: m.Spec.To.Name,
									Port: port,
								},
							},
						}},
					},
				},
			}},
		},
	}
	// Set Route instance as the owner and controller
	ctrl.SetControllerReference(m, ing, r.Scheme)
	return ing, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&routev1.Route{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
