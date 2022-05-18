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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	selector, err := metav1.LabelSelectorAsSelector(&res.Spec.EntitySelector)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.List(ctx, entities, client.MatchingLabelsSelector{Selector: selector}); err != nil {
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

	if err := r.Get(ctx, client.ObjectKeyFromObject(problem), problem); err != nil {
		return ctrl.Result{}, err
	}
	if problem.Generation == problem.Status.ObservedGeneration {
		res.Status.Solution = problem.Status.Solution
		res.Status.Error = problem.Status.Error
	}
	res.Status.ObservedGeneration = res.Generation

	return ctrl.Result{}, r.Status().Update(ctx, res)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResolutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuberdepv1.Resolution{}).
		Owns(&kuberdepv1.Problem{}).
		Watches(&source.Kind{Type: &kuberdepv1.Entity{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			l := mgr.GetLogger()
			entity := object.(*kuberdepv1.Entity)
			resolutions, err := mapEntityToResolutions(context.Background(), r.Client, *entity)
			if err != nil {
				l.Error(err, "map entity to resolutions")
				return nil
			}
			var reqs []reconcile.Request
			for _, resolution := range resolutions {
				reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&resolution)})
			}
			return reqs
		})).
		Watches(&source.Kind{Type: &kuberdepv1.ConstraintDefinition{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			l := mgr.GetLogger()
			cd := object.(*kuberdepv1.ConstraintDefinition)
			resolutions, err := mapConstraintDefinitionToResolutions(context.Background(), r.Client, *cd)
			if err != nil {
				l.Error(err, "map constraint definition to resolutions")
				return nil
			}
			var reqs []reconcile.Request
			for _, resolution := range resolutions {
				reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&resolution)})
			}
			return reqs
		})).
		Complete(r)
}

func mapConstraintDefinitionToEntities(ctx context.Context, cl client.Client, cd kuberdepv1.ConstraintDefinition) ([]kuberdepv1.Entity, error) {
	entities := &kuberdepv1.EntityList{}
	if err := cl.List(ctx, entities); err != nil {
		return nil, err
	}

	var out []kuberdepv1.Entity
	for _, entity := range entities.Items {
		for _, c := range entity.Spec.Constraints {
			if c.Type == cd.Spec.ID {
				out = append(out, entity)
				break
			}
		}
	}
	return out, nil
}

func mapEntityToResolutions(ctx context.Context, cl client.Client, entity kuberdepv1.Entity) ([]kuberdepv1.Resolution, error) {
	resolutions := &kuberdepv1.ResolutionList{}
	if err := cl.List(ctx, resolutions); err != nil {
		return nil, err
	}

	var out []kuberdepv1.Resolution
	for _, resolution := range resolutions.Items {
		ls, err := metav1.LabelSelectorAsSelector(&resolution.Spec.EntitySelector)
		if err != nil {
			return nil, err
		}

		if ls.Matches(labels.Set(entity.Labels)) {
			out = append(out, resolution)
			continue
		}
	}
	return out, nil
}

func mapConstraintDefinitionToResolutions(ctx context.Context, cl client.Client, cd kuberdepv1.ConstraintDefinition) ([]kuberdepv1.Resolution, error) {
	entities, err := mapConstraintDefinitionToEntities(ctx, cl, cd)
	if err != nil {
		return nil, err
	}
	var out []kuberdepv1.Resolution
	for _, entity := range entities {
		resolutions, err := mapEntityToResolutions(ctx, cl, entity)
		if err != nil {
			return nil, err
		}
		out = append(out, resolutions...)
	}
	return out, nil
}
