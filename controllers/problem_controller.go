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
	"github.com/joelanford/kuberdep/pkg/solver"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kuberdepv1 "github.com/joelanford/kuberdep/api/v1"
)

// ProblemReconciler reconciles a Problem object
type ProblemReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kuberdep.io,resources=problems,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuberdep.io,resources=problems/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuberdep.io,resources=problems/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProblemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.V(1).Info("starting reconciliation")
	defer l.V(1).Info("ending reconciliation")

	prob := &kuberdepv1.Problem{}
	if err := r.Get(ctx, req.NamespacedName, prob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var vars []solver.Variable
	for _, entity := range prob.Spec.Entities {
		v := &variable{id: entity.ID}
		for _, c := range entity.Constraints {
			if c.Mandatory != nil && *c.Mandatory {
				v.constraints = append(v.constraints, solver.Mandatory())
			}
			if c.Prohibited != nil && *c.Prohibited {
				v.constraints = append(v.constraints, solver.Prohibited())
			}
			if c.DependsOnOneOf != nil {
				v.constraints = append(v.constraints, solver.Dependency(asSolverIDs(c.DependsOnOneOf...)...))
			}
			if c.ConflictsWith != nil {
				v.constraints = append(v.constraints, solver.Conflict(solver.IdentifierFromString(*c.ConflictsWith)))
			}
			if c.AtMostOf != nil {
				v.constraints = append(v.constraints, solver.AtMost(c.AtMostOf.Number, asSolverIDs(c.AtMostOf.IDs...)...))
			}
		}
		vars = append(vars, v)
	}

	s, err := solver.New(solver.WithInput(vars))
	if err != nil {
		prob.Status.Solution = nil
		prob.Status.ObservedGeneration = prob.Generation
		prob.Status.Error = err.Error()
		return ctrl.Result{}, r.Status().Update(ctx, prob)
	}
	solution, err := s.Solve(ctx)
	if err != nil {
		prob.Status.Solution = nil
		prob.Status.ObservedGeneration = prob.Generation
		prob.Status.Error = err.Error()
		return ctrl.Result{}, r.Status().Update(ctx, prob)
	}

	prob.Status.Solution = asIDStrings(solution...)
	prob.Status.ObservedGeneration = prob.Generation
	prob.Status.Error = ""

	return ctrl.Result{}, r.Status().Update(ctx, prob)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuberdepv1.Problem{}).
		Complete(r)
}

type variable struct {
	id          string
	constraints []solver.Constraint
}

func (v *variable) Identifier() solver.Identifier {
	return solver.IdentifierFromString(v.id)
}

func (v *variable) Constraints() []solver.Constraint {
	return v.constraints
}

func asSolverIDs(ids ...string) []solver.Identifier {
	solverIDs := make([]solver.Identifier, len(ids))
	for i, id := range ids {
		solverIDs[i] = solver.IdentifierFromString(id)
	}
	return solverIDs
}

func asIDStrings(vars ...solver.Variable) []string {
	ids := make([]string, len(vars))
	for i, v := range vars {
		ids[i] = v.Identifier().String()
	}
	return ids
}
