/*
Copyright 2023 Aapeli <aapeli.nian@gmail.com>.

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
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	controllerutils "github.com/frp-sigs/frp-provisioner/pkg/utils/controller"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/storage/names"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const defaultBaseName = "frp-client"

// ServiceReconciler reconciles a FrpServer object
type ServiceReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Options *config.ManagerOptions
}

func (r *ServiceReconciler) getOwnedPods(ctx context.Context, instance *v1.Service) ([]*v1.Pod, []*v1.Pod, error) {
	logger := log.FromContext(ctx)
	podList := &v1.PodList{}
	opts := &client.ListOptions{
		Namespace: instance.Namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			v1beta1.LabelServiceNameKey:   instance.Name,
			v1beta1.LabelControllerUidKey: string(instance.UID),
		}),
	}
	if err := r.List(ctx, podList, opts); err != nil {
		logger.WithValues("namespace", instance.Namespace).Error(err, "unable get pod list")
		return nil, nil, err
	}
	var activePods, inactivePods []*v1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		if controllerutils.IsPodActive(pod) {
			activePods = append(activePods, pod)
		} else {
			inactivePods = append(inactivePods, pod)
		}
	}
	return activePods, inactivePods, nil
}

func (r *ServiceReconciler) generatePod(ctx context.Context, owner *v1.Service) (*v1.Pod, error) {
	logger := log.FromContext(ctx)
	pod := &v1.Pod{}
	if err := yaml.Unmarshal([]byte(r.Options.PodTemplate), pod); err != nil {
		logger.Error(err, "unable parse yaml from pod template", "template", r.Options.PodTemplate)
		return nil, fmt.Errorf("unable parse yaml from pod template, err: %w", err)
	}
	if pod.GetLabels() == nil {
		pod.SetLabels(make(map[string]string))
	}
	baseName := defaultBaseName
	if pod.GetName() != "" {
		baseName = pod.GetName()
	}
	pod.SetNamespace(owner.Namespace)
	pod.SetName(names.SimpleNameGenerator.GenerateName(baseName + "-" + owner.Name))
	if err := controllerutil.SetControllerReference(owner, pod, r.Scheme); err != nil {
		logger.Error(err, "can't set Pod owner reference", "namespace", pod.GetNamespace(), "name", pod.GetName())
		return nil, fmt.Errorf("can't set Pod '%v/%v' owner reference: %w", pod.GetNamespace(), pod.GetName(), err)
	}
	pod.Labels[v1beta1.LabelServiceNameKey] = owner.Name
	pod.Labels[v1beta1.LabelControllerUidKey] = string(owner.UID)
	return pod, nil
}

//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=services/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	instance := &v1.Service{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// skip deleted object
			logger.Info("service has been deleted", "request", req.String())
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable get service by name", "request", req.String())
		return ctrl.Result{}, err
	}
	activePods, inactivePods, err := r.getOwnedPods(ctx, instance)
	if err != nil {
		logger.Error(err, "unable get owner pods for service", "request", req.String())
		return ctrl.Result{}, err
	}
	claimedPods, err := r.claimPods(instance, activePods)
	if err != nil {
		logger.Error(err, "unable get claimed pods for service", "request", req.String())
		return ctrl.Result{}, err
	}
	errsList := make([]error, 0)
	// kill all inactive pods
	for _, pod := range inactivePods {
		if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "unable delete pod", "podName", pod.GetName())
			errsList = append(errsList, err)
		}
	}
	if len(errsList) != 0 {
		return ctrl.Result{}, utilerrors.NewAggregate(errsList)
	}
	// clean for delete service or service type is not LoadBalancer
	if instance.Spec.Type != v1.ServiceTypeLoadBalancer || len(instance.Annotations) == 0 ||
		instance.Annotations[v1beta1.AnnotationFrpServerNameKey] == "" || instance.DeletionTimestamp != nil {
		for _, pod := range claimedPods {
			if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "unable delete pod for service", "podName", pod.GetName(), "service", req)
				errsList = append(errsList, fmt.Errorf("unable delete pod '%s', err: %w", req.String(), err))
			}
		}
		instance.Finalizers = lo.Without(instance.Finalizers, v1beta1.FinalizerName)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "unable remove finalizers for service", "service", req.String())
			errsList = append(errsList, fmt.Errorf("unable remove finalizers for service '%s', err: %w", req.String(), err))
		}
		return ctrl.Result{}, utilerrors.NewAggregate(errsList)
	}
	// add finalizer for current service
	if !lo.Contains(instance.Finalizers, v1beta1.FinalizerName) {
		instance.Finalizers = append(instance.Finalizers, v1beta1.FinalizerName)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "unable add finalizers for service", "service", req.String())
			return ctrl.Result{}, fmt.Errorf("unable add finalizers for service '%s', err: %w", req.String(), err)
		}
	}
	if len(claimedPods) == 0 {
		pod, err := r.generatePod(ctx, instance)
		if err != nil {
			logger.Error(err, "unable generate pod from podTemplate")
			return ctrl.Result{}, fmt.Errorf("unable generate pod from podTemplate, err: %w", err)
		}
		if err := r.Create(ctx, pod); err != nil {
			logger.Error(err, "unable create frp pod by template", "pod", fmt.Sprintf("%+v", pod))
			return ctrl.Result{}, fmt.Errorf("unable create frp pod '%+v',err: %w", pod, err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) scheduleServer(ctx context.Context, instance *v1.Service) (*v1beta1.FrpServer, error) {
	logger := log.FromContext(ctx)
	if len(instance.Annotations) == 0 {
		return nil, fmt.Errorf("please set annotations.%s to assign frp server", v1beta1.AnnotationFrpServerNameKey)
	}
	serverName, ok := instance.Annotations[v1beta1.AnnotationFrpServerNameKey]
	if !ok || serverName == "" {
		return nil, fmt.Errorf("please set annotations.%s to assign frp server", v1beta1.AnnotationFrpServerNameKey)
	}
	objectKey := client.ObjectKey{Name: serverName}
	server := &v1beta1.FrpServer{}
	if err := r.Get(ctx, objectKey, server); err != nil {
		logger.WithValues("request", objectKey.String()).Error(err, "unable get v1beta1.FrpServer by name")
		return nil, err
	}
	return server, nil
}

func (r *ServiceReconciler) getFrpServers(ctx context.Context, instance v1.Service) ([]*v1beta1.FrpServer, []*v1beta1.FrpServer, error) {
	logger := log.FromContext(ctx)
	serverList := &v1beta1.FrpServerList{}
	if err := r.List(ctx, serverList); err != nil {
		logger.WithValues("namespace", instance.Namespace).Error(err, "Unable get frpserver list")
		return nil, nil, err
	}
	var activeServers, inactiveServers []*v1beta1.FrpServer
	for i := range serverList.Items {
		srv := &serverList.Items[i]
		if controllerutils.IsFrpServerActive(srv) {
			activeServers = append(activeServers, srv)
		} else {
			inactiveServers = append(inactiveServers, srv)
		}
	}
	return activeServers, inactiveServers, nil
}

func (r *ServiceReconciler) claimPods(instance *v1.Service, pods []*v1.Pod) ([]*v1.Pod, error) {
	selector := labels.SelectorFromSet(labels.Set{
		v1beta1.LabelServiceNameKey:   instance.Name,
		v1beta1.LabelControllerUidKey: string(instance.UID),
	})
	mgr, err := controllerutils.NewRefManager(r.Client, selector, instance, r.Scheme)
	if err != nil {
		return nil, err
	}
	selected := make([]metav1.Object, len(pods))
	for i, pod := range pods {
		selected[i] = pod
	}
	claimed, err := mgr.ClaimOwnedObjects(selected)
	if err != nil {
		return nil, err
	}
	claimedPods := make([]*v1.Pod, len(claimed))
	for i, pod := range claimed {
		claimedPods[i] = pod.(*v1.Pod)
	}
	return claimedPods, nil
}

// SetupWithManager set up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Service{}).
		Owns(&v1.Pod{}).
		Complete(r)
}
