/*
Copyright 2023 The Frp Sig Authors.

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

package controller

import (
	"context"
	"fmt"
	frpv1beta1 "github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/utils/frp"
	"github.com/frp-sigs/frp-provisioner/pkg/utils/validate"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// FrpServerReconciler reconciles a FrpServer object
type FrpServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=frp.gofrp.io,resources=frpservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=frp.gofrp.io,resources=frpservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=frp.gofrp.io,resources=frpservers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *FrpServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	obj := frpv1beta1.FrpServer{}

	err := r.Get(ctx, req.NamespacedName, &obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "get resource object failed.", "request", req.String())
		return ctrl.Result{}, nil
	}

	// Set phase to FrpServerPhasePending and wait next Reconcile
	if obj.Status.Phase == frpv1beta1.FrpServerPhaseUnknown {
		obj.Status.Phase = frpv1beta1.FrpServerPhasePending
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, &obj)})
	}

	commonConfig, err := frp.GenClientCommonConfig(ctx, r.Client, &obj)
	if err != nil {
		logger.Error(err, "Error generate frp config from resource object")
		meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               "Initialized",
			Status:             metav1.ConditionTrue,
			Reason:             frpv1beta1.ReasonGenerateConfigFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to generate frp config: %s", err.Error()),
		})
		obj.Status.Phase = frpv1beta1.FrpServerPhaseUnhealthy
		obj.Status.Reason = fmt.Sprintf("unable to generate frp config: %s", err.Error())
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, &obj)})
	}

	err = validate.ValidateClientCommonConfig(ctx, commonConfig)
	if err != nil {
		logger.Error(err, "Invalid frp config from resource object")
		meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               "Initialized",
			Status:             metav1.ConditionTrue,
			Reason:             frpv1beta1.ReasonInitializeFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("Invalid frp config: %s", err.Error()),
		})
		obj.Status.Phase = frpv1beta1.FrpServerPhaseUnhealthy
		obj.Status.Reason = fmt.Sprintf("Invalid frp config: %s", err.Error())
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, &obj)})
	}

	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               "Initialized",
		Status:             metav1.ConditionTrue,
		Reason:             frpv1beta1.ReasonInitialized,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Message:            "FrpServer is healthy",
	})
	obj.Status.Phase = frpv1beta1.FrpServerPhaseHealthy
	obj.Status.Reason = "FrpServer is healthy"

	return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, &obj)})
}

// SetupWithManager sets up the controller with the Manager.
func (r *FrpServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frpv1beta1.FrpServer{}).
		Complete(r)
}
