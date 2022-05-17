/*
Copyright 2022.

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
	"github.com/joelanford/kuberdep/pkg/plugin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"

	kuberdepv1 "github.com/joelanford/kuberdep/api/v1"
)

// ResolutionReconciler reconciles a Resolution object
type ResolutionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kuberdep.io,resources=resolutions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuberdep.io,resources=resolutions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuberdep.io,resources=resolutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ResolutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.V(1).Info("starting reconciliation")
	defer l.V(1).Info("ending reconciliation")

	res := &kuberdepv1.Resolution{}
	if err := r.Get(ctx, req.NamespacedName, res); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	entities := &kuberdepv1.EntityList{}
	if err := r.List(ctx, entities); err != nil {
		return ctrl.Result{}, err
	}

	plugins, err := plugin.New(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	sort.Slice(entities.Items, func(a, b int) bool {
		return entities.Items[a].Name < entities.Items[b].Name
	})

	pluginEntities, err := plugin.ConvertEntities(entities.Items...)
	if err != nil {
		return ctrl.Result{}, err
	}

	var problemEntities []kuberdepv1.ProblemEntity
	for i, entity := range pluginEntities {
		items := make([]plugin.Entity, 0, len(pluginEntities)-1)
		items = append(items, pluginEntities[:i]...)
		items = append(items, pluginEntities[i+1:]...)
		var allConstraints []kuberdepv1.Constraint
		for _, constraint := range entity.Constraints {
			pi := plugin.Input{
				Config:  constraint.Value,
				Subject: entity,
				Items:   items,
			}
			constraints, err := plugins.Exec(constraint.Type, pi)

			if err != nil {
				// TODO: update status
				return ctrl.Result{}, fmt.Errorf("exec plugin: %v", err)
			}
			allConstraints = append(allConstraints, constraints...)
		}
		problemEntities = append(problemEntities, kuberdepv1.ProblemEntity{
			ID:          entity.ID,
			Constraints: allConstraints,
		})
	}

	problem := &kuberdepv1.Problem{
		ObjectMeta: metav1.ObjectMeta{
			Name:      res.Name,
			Namespace: res.Namespace,
		},
		Spec: kuberdepv1.ProblemSpec{
			Entities: problemEntities,
		},
	}
	sort.Slice(problemEntities, func(a, b int) bool {
		return problemEntities[a].ID < problemEntities[b].ID
	})

	if err := controllerutil.SetControllerReference(res, problem, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, problem); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Get(ctx, client.ObjectKeyFromObject(problem), problem); err != nil {
				return ctrl.Result{}, err
			}
			problem.Spec.Entities = problemEntities
			if err := r.Update(ctx, problem); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResolutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuberdepv1.Resolution{}).
		Owns(&kuberdepv1.Problem{}).
		Complete(r)
}
