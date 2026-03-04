// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconciler

import (
	"context"
	"errors"
	"sort"

	"github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PortfolioReconciler reconciles Portfolio objects.
// It lists Workspaces matching the portfolio's label selector and updates
// the portfolio status with workspace count and names.
type PortfolioReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// Verify PortfolioReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &PortfolioReconciler{}

// Reconcile implements the reconciliation loop for Portfolio resources.
func (r *PortfolioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("portfolio", req.NamespacedName)

	// 1. Fetch Portfolio CR
	var portfolio v1alpha1.Portfolio
	if err := r.Get(ctx, req.NamespacedName, &portfolio); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Portfolio not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Join(err, errors.New("failed to get portfolio"))
	}

	// 2. Check suspend
	if portfolio.Spec.Suspend {
		setPortfolioCondition(&portfolio, ConditionStalled, metav1.ConditionTrue, "Suspended", "Portfolio is suspended")
		setPortfolioCondition(&portfolio, ConditionReconciling, metav1.ConditionFalse, "Suspended", "Portfolio is suspended")
		portfolio.Status.ObservedGeneration = portfolio.Generation
		if err := r.Status().Update(ctx, &portfolio); err != nil {
			log.Error(err, "Failed to update Portfolio status for suspend")
			return ctrl.Result{}, errors.Join(err, errors.New("failed to update portfolio status"))
		}
		log.Info("Portfolio suspended")
		return ctrl.Result{}, nil
	}

	// 3. Handle nil selector: no workspaces match
	if portfolio.Spec.Selector == nil {
		portfolio.Status.WorkspaceCount = 0
		portfolio.Status.WorkspaceNames = nil
		portfolio.Status.ObservedGeneration = portfolio.Generation
		setPortfolioCondition(&portfolio, ConditionReady, metav1.ConditionTrue, "NoSelector", "No selector defined, workspace count is zero")
		setPortfolioCondition(&portfolio, ConditionStalled, metav1.ConditionFalse, "NotStalled", "Portfolio is not stalled")
		setPortfolioCondition(&portfolio, ConditionReconciling, metav1.ConditionFalse, "ReconcileComplete", "Reconciliation complete")
		if err := r.Status().Update(ctx, &portfolio); err != nil {
			log.Error(err, "Failed to update Portfolio status for nil selector")
			return ctrl.Result{}, errors.Join(err, errors.New("failed to update portfolio status"))
		}
		return ctrl.Result{}, nil
	}

	// 4. Convert label selector and list matching Workspaces
	sel, err := metav1.LabelSelectorAsSelector(portfolio.Spec.Selector)
	if err != nil {
		setPortfolioCondition(&portfolio, ConditionStalled, metav1.ConditionTrue, "InvalidSelector", "Failed to parse label selector")
		setPortfolioCondition(&portfolio, ConditionReady, metav1.ConditionFalse, "InvalidSelector", err.Error())
		portfolio.Status.ObservedGeneration = portfolio.Generation
		_ = r.Status().Update(ctx, &portfolio)
		return ctrl.Result{}, errors.Join(err, errors.New("failed to convert label selector"))
	}

	var workspaceList v1alpha1.WorkspaceList
	if err := r.List(ctx, &workspaceList,
		client.InNamespace(req.Namespace),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return ctrl.Result{}, errors.Join(err, errors.New("failed to list workspaces"))
	}

	// 5. Update status
	names := make([]string, 0, len(workspaceList.Items))
	for i := range workspaceList.Items {
		names = append(names, workspaceList.Items[i].Name)
	}
	sort.Strings(names)

	portfolio.Status.WorkspaceCount = len(workspaceList.Items)
	portfolio.Status.WorkspaceNames = names
	portfolio.Status.ObservedGeneration = portfolio.Generation

	setPortfolioCondition(&portfolio, ConditionReady, metav1.ConditionTrue, "ReconcileComplete", "Reconciliation complete")
	setPortfolioCondition(&portfolio, ConditionStalled, metav1.ConditionFalse, "NotStalled", "Portfolio is not stalled")
	setPortfolioCondition(&portfolio, ConditionReconciling, metav1.ConditionFalse, "ReconcileComplete", "Reconciliation complete")

	if err := r.Status().Update(ctx, &portfolio); err != nil {
		log.Error(err, "Failed to update Portfolio status")
		return ctrl.Result{}, errors.Join(err, errors.New("failed to update portfolio status"))
	}

	log.Info("Reconciliation complete", "workspaceCount", portfolio.Status.WorkspaceCount)
	return ctrl.Result{}, nil
}

// setPortfolioCondition sets a condition on the Portfolio status.
func setPortfolioCondition(p *v1alpha1.Portfolio, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&p.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: p.Generation,
		Reason:             reason,
		Message:            message,
	})
}
