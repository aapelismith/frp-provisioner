package ingress

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsInformerV1 "k8s.io/client-go/informers/apps/v1"
	coreInformerV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	appsListerV1 "k8s.io/client-go/listers/apps/v1"
	coreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"kunstack.com/pharos/pkg/safe"
	"kunstack.com/pharos/pkg/types"
	"kunstack.com/pharos/pkg/utils"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	controllerId         = "lb-controller"
	controllerIdLabelKey = "loadbalancer.kunstack.com/id"
	daemonSetPrefix      = "lb-"
)

type Controller struct {
	ctx                context.Context
	options            *Options
	serviceHasSynced   func() bool
	daemonSetHasSynced func() bool
	daemonSetHelper    *appsV1.DaemonSet
	kubeClient         kubernetes.Interface
	serviceLister      coreListerV1.ServiceLister
	daemonSetLister    appsListerV1.DaemonSetLister
	eventRecord        record.EventRecorder
	serviceQueue       workqueue.RateLimitingInterface
	daemonSetQueue     workqueue.RateLimitingInterface
}

// Run the load balancing controller, observe the changes of all services in k8s,
// and reload the proxy daemon configuration file in due course
func (c *Controller) Run(stopChan <-chan struct{}, serviceWorkers int) {
	var (
		wg     = new(sync.WaitGroup)
		logger = logr.FromContextOrDiscard(c.ctx)
	)

	defer runtime.HandleCrash()

	logger.V(1).Info("Starting loadbalancer controller")
	defer logger.V(1).Info("Shutting loadbalancer controller")

	// Wait for the cache to be synchronized
	if !cache.WaitForCacheSync(stopChan, c.serviceHasSynced, c.daemonSetHasSynced) {
		return
	}

	// Execute synchronization process
	for i := 0; i < serviceWorkers; i++ {
		wg.Add(2)
		safe.Go(func() {
			defer wg.Done()
			wait.Until(c.serviceWorker, time.Second, stopChan)
		})

		safe.Go(func() {
			defer wg.Done()
			wait.Until(c.daemonSetWorker, time.Second, stopChan)
		})
	}

	<-stopChan
	// Close all queues
	c.serviceQueue.ShutDown()
	c.daemonSetQueue.ShutDown()
	// Wait for all goroutines to exit
	wg.Wait()
}

// serviceWorker runs a serviceWorker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Controller) serviceWorker() {
	for c.processNextServiceWorkItem() {
	}
}

// daemonSetWorker runs a daemonSetWorker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Controller) daemonSetWorker() {
	for c.processNextDaemonSetWorkItem() {
	}
}

func (c *Controller) processNextServiceWorkItem() bool {
	key, quit := c.serviceQueue.Get()
	if quit {
		return false
	}
	defer c.serviceQueue.Done(key)

	if err := c.syncService(key.(string)); err == nil {
		c.serviceQueue.Forget(key)
		return true
	}

	//A single key requeue more than 5 times and then gives up
	if c.serviceQueue.NumRequeues(key) > 5 {
		c.serviceQueue.Forget(key)
	}

	c.serviceQueue.AddRateLimited(key)
	return true
}

func (c *Controller) processNextDaemonSetWorkItem() bool {
	key, quit := c.daemonSetQueue.Get()
	if quit {
		return false
	}
	defer c.daemonSetQueue.Done(key)

	if err := c.syncDaemonSet(key.(string)); err == nil {
		c.daemonSetQueue.Forget(key)
		return true
	}

	//A single key requeue more than 5 times and then gives up
	if c.daemonSetQueue.NumRequeues(key) > 5 {
		c.daemonSetQueue.Forget(key)
	}

	c.daemonSetQueue.AddRateLimited(key)
	return true
}

func (c *Controller) applyLoadBalancer(key string) (err error) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()
	logger := logr.FromContextOrDiscard(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "couldn't split meta namespace key %s, got: %v", key, err)
		return err
	}

	svcShared, err := c.serviceLister.Services(namespace).Get(name)
	if err != nil {
		logger.Error(err, "unable get service %s, got: %v", key, err)
		return err
	}

	appName := daemonSetPrefix + name
	svc := svcShared.DeepCopy()

	defer func() {
		if err != nil {
			c.eventRecord.Eventf(
				svc, v1.EventTypeWarning, "LoadBalancerSyncFailed",
				"Load balancer synchronization failed, got: %v", err,
			)
			return
		}
	}()

	daemonSetHelper := c.daemonSetHelper.DeepCopy()

	podContainers := make([]v1.Container, 0, len(svc.Spec.Ports))

	for _, port := range svc.Spec.Ports {
		container := daemonSetHelper.Spec.Template.Spec.Containers[0].DeepCopy()
		container.Name = fmt.Sprintf("%s-%d", svc.Name, port.Port)

		container.Ports = append(
			container.Ports,
			v1.ContainerPort{ContainerPort: port.Port, HostPort: port.Port},
		)

		envs := make([]v1.EnvVar, 0, len(container.Env))

		// Old environment variables that need to be deleted
		skips := []string{"SRC_PORT", "DEST_PROTO", "DEST_PORT", "DEST_IP"}

		// Skip unnecessary environment variables and keep other environment variables
		for _, env := range container.Env {
			if utils.StringIn(skips, env.Name) {
				continue
			}

			envs = append(envs, env)
		}

		envs = append(
			envs,
			v1.EnvVar{Name: "SRC_PORT", Value: strconv.Itoa(int(port.Port))},
			v1.EnvVar{Name: "DEST_PROTO", Value: string(port.Protocol)},
			v1.EnvVar{Name: "DEST_PORT", Value: strconv.Itoa(int(port.Port))},
			v1.EnvVar{Name: "DEST_IP", Value: svc.Spec.ClusterIP},
		)

		container.Env = envs

		if container.SecurityContext == nil {
			container.SecurityContext = &v1.SecurityContext{}
		}

		if container.SecurityContext.Capabilities == nil {
			container.SecurityContext.Capabilities = &v1.Capabilities{}
		}

		addCaps := make([]v1.Capability, 0, len(container.SecurityContext.Capabilities.Add))

		for _, add := range container.SecurityContext.Capabilities.Add {
			if add == "NET_ADMIN" {
				continue
			}

			addCaps = append(addCaps, add)
		}

		addCaps = append(addCaps, "NET_ADMIN")

		container.SecurityContext.Capabilities.Add = addCaps

		podContainers = append(podContainers, *container)
	}

	daemonSetHelper.Spec.Template.Spec.Containers = podContainers

	daemonSetHelper.ObjectMeta.Labels["app"] = appName
	daemonSetHelper.ObjectMeta.Labels[controllerIdLabelKey] = controllerId

	daemonSetHelper.ObjectMeta.OwnerReferences = []metaV1.OwnerReference{{
		APIVersion: v1.SchemeGroupVersion.String(),
		Kind:       "Service",
		Name:       svc.Name,
		UID:        svc.UID,
		Controller: types.Bool(true),
	}}

	_, err = c.kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, appName, metaV1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Unable get daemonset by name %s", appName)
		return err
	}

	if errors.IsNotFound(err) {
		_, err = c.kubeClient.AppsV1().DaemonSets(namespace).Create(ctx, daemonSetHelper, metaV1.CreateOptions{})
		if err != nil {
			logger.Error(err, "Unable create daemonset got: %v", err)
			return err
		}
		return nil
	}

	_, err = c.kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonSetHelper, metaV1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "unable update DaemonSet got: %v", err)
		return err
	}

	var flag bool

	for _, ingres := range svc.Status.LoadBalancer.Ingress {
		if c.options.DomainSuffix != "" && strings.HasSuffix(ingres.Hostname, c.options.DomainSuffix) {
			flag = true
			break
		}
	}

	if !flag {
		svc.Status.LoadBalancer.Ingress = append(
			svc.Status.LoadBalancer.Ingress,
			v1.LoadBalancerIngress{Hostname: uuid.New().String() + c.options.DomainSuffix},
		)

		_, err = c.kubeClient.CoreV1().Services(namespace).UpdateStatus(ctx, svc, metaV1.UpdateOptions{})
		if err != nil {
			logger.Error(err, "Unable update service status, got: %v", err)
			return err
		}
	}

	return nil
}

func (c *Controller) deleteLoadBalancer(key string) error {
	// Wait up to one minute, cancel the context after one minute
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()

	logger := logr.FromContextOrDiscard(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.V(1).Error(err, "couldn't split meta namespace key %s,got: %v", key, err)
		return err
	}

	err = c.kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, daemonSetPrefix+name, metaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Unable to delete DaemonSet %v, got: %v", key, err)
		return err
	}

	svcShared, err := c.serviceLister.Services(namespace).Get(name)
	if err != nil {
		logger.Error(err, "Unable get service by name %s", name)
		return err
	}

	svc := svcShared.DeepCopy()

	cleanIngress := make([]v1.LoadBalancerIngress, 0, len(svc.Status.LoadBalancer.Ingress))

	// Modify the status of svc
	for _, ingres := range svc.Status.LoadBalancer.Ingress {
		if strings.HasSuffix(ingres.Hostname, c.options.DomainSuffix) {
			continue
		}
		cleanIngress = append(cleanIngress, ingres)
	}

	svc.Status.LoadBalancer.Ingress = cleanIngress

	_, err = c.kubeClient.CoreV1().Services(namespace).UpdateStatus(ctx, svc, metaV1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Unable update service statue")
		return err
	}
	return nil
}

func (c *Controller) syncDaemonSet(key string) error {
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()
	logger := logr.FromContextOrDiscard(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "couldn't split meta namespace key %s,got: %v", key, err)
		return err
	}

	ds, err := c.daemonSetLister.DaemonSets(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "couldn't get DaemonSet by key %s got: %v", key, err)
		return err
	}

	// 如果ds被人删了, 将ds对应的svc重新入队列，准备创建新的ds
	if errors.IsNotFound(err) {
		svcName := strings.TrimPrefix(name, daemonSetPrefix)
		c.serviceQueue.Add(fmt.Sprintf("%s/%s", namespace, svcName))
		return nil
	}

	ownerReference := metaV1.GetControllerOf(ds)

	svc, err := c.serviceLister.Services(namespace).Get(ownerReference.Name)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Unable get service by name %s", ownerReference.Name)
		return err
	}

	// svc is deleted so directly delete DaemonSet
	if errors.IsNotFound(err) {
		return c.kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
	}

	// DaemonSet has been modified, we don’t want it to be modified manually,
	// re-enter svc into the queue and start to rebuild DaemonSet
	c.addService(svc)
	return nil
}

func (c *Controller) syncService(key string) error {
	// Wait up to one minute, cancel the context after one minute
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()
	logger := logr.FromContextOrDiscard(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "couldn't split meta namespace key %s,got: %v", key, err)
		return err
	}

	svc, err := c.serviceLister.Services(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "couldn't get service by key %s got: %v", key, err)
		return err
	}

	if errors.IsNotFound(err) {
		return c.deleteLoadBalancer(key)
	}

	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return c.deleteLoadBalancer(key)
	}

	// 更新/创建操作
	return c.applyLoadBalancer(key)
}

func (c *Controller) addDaemonSet(obj interface{}) {
	ds := obj.(*appsV1.DaemonSet)

	if ds.Labels == nil {
		return
	}

	val, _ := ds.Labels[controllerIdLabelKey]
	if val != controllerId {
		return
	}

	ownerReference := metaV1.GetControllerOf(ds)
	if ownerReference == nil {
		return
	}

	if ownerReference.Controller != nil && !*ownerReference.Controller {
		return
	}

	if ownerReference.Kind != "Service" {
		return
	}

	if ds.DeletionTimestamp != nil {
		// on a restart of the controller, it's possible a new DaemonSet shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deleteDaemonSet(ds)
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(ds)
	if err != nil {
		return
	}
	c.daemonSetQueue.Add(key)
}

func (c *Controller) deleteDaemonSet(obj interface{}) {
	// Wait up to one minute, cancel the context after one minute
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()
	logger := logr.FromContextOrDiscard(ctx)

	ds, ok := obj.(*appsV1.DaemonSet)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			logger.V(1).Info("couldn't get object from tombstone %+v", obj)
			return
		}
		ds, ok = tombstone.Obj.(*appsV1.DaemonSet)
		if !ok {
			logger.V(1).Info("tombstone contained object that is not a service %+v", obj)
			return
		}
	}

	if ds.Labels == nil {
		return
	}

	val, _ := ds.Labels[controllerIdLabelKey]
	if val != controllerId {
		return
	}

	ownerReference := metaV1.GetControllerOf(ds)
	if ownerReference == nil {
		return
	}

	if ownerReference.Controller != nil && !*ownerReference.Controller {
		return
	}

	if ownerReference.Kind != "Service" {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(ds)
	if err != nil {
		return
	}
	c.daemonSetQueue.Add(key)
}

func (c *Controller) updateDaemonSet(_, obj interface{}) {
	ds := obj.(*appsV1.DaemonSet)

	if ds.Labels == nil {
		return
	}

	val, _ := ds.Labels[controllerIdLabelKey]
	if val != controllerId {
		return
	}

	ownerReference := metaV1.GetControllerOf(ds)
	if ownerReference == nil {
		return
	}

	if ownerReference.Controller != nil && !*ownerReference.Controller {
		return
	}

	if ownerReference.Kind != "Service" {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(ds)
	if err != nil {
		return
	}
	c.daemonSetQueue.Add(key)
}

func (c *Controller) addService(obj interface{}) {
	svc := obj.(*v1.Service)
	if svc.DeletionTimestamp != nil {
		// on a restart of the controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deleteService(svc)
		return
	}

	if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		key, err := cache.MetaNamespaceKeyFunc(svc)
		if err != nil {
			return
		}
		c.serviceQueue.Add(key)
	}
}

func (c *Controller) updateService(old, obj interface{}) {
	// Wait up to one minute, cancel the context after one minute
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()

	logger := logr.FromContextOrDiscard(ctx)

	svc := obj.(*v1.Service)
	if svc.DeletionTimestamp != nil {
		// on a restart of the controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deleteService(svc)
		return
	}

	oldSvc := old.(*v1.Service)

	// If the three types of port/type/clusterIp have not changed, ignore the other content.
	if oldSvc.Spec.Type == svc.Spec.Type && reflect.DeepEqual(svc.Spec.Ports, oldSvc.Spec.Ports) && svc.Spec.ClusterIP == oldSvc.Spec.ClusterIP {
		return
	}

	// The service is put into the work queue if it was or is currently of LoadBalancer type
	if svc.Spec.Type == v1.ServiceTypeLoadBalancer || oldSvc.Spec.Type == v1.ServiceTypeLoadBalancer {
		key, err := cache.MetaNamespaceKeyFunc(svc)
		if err != nil {
			logger.Error(err, "couldn't get meta namespace key for object %+v, got: %v", svc, err)
			return
		}
		c.serviceQueue.Add(key)
	}
}

func (c *Controller) deleteService(obj interface{}) {
	// Wait up to one minute, cancel the context after one minute
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()

	logger := logr.FromContextOrDiscard(ctx)

	svc, ok := obj.(*v1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			logger.V(1).Info("couldn't get object from tombstone %+v", obj)
			return
		}
		svc, ok = tombstone.Obj.(*v1.Service)
		if !ok {
			logger.V(1).Info("tombstone contained object that is not a service %+v", obj)
			return
		}
	}

	if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		key, err := cache.MetaNamespaceKeyFunc(svc)
		if err != nil {
			return
		}
		c.serviceQueue.Add(key)
	}
}

// NewController creates a new service controller that keeps the relevant pods
// in sync with their corresponding service objects.
func NewController(ctx context.Context, opt *Options, serviceInformer coreInformerV1.ServiceInformer, daemonSetInformer appsInformerV1.DaemonSetInformer, kubeClient kubernetes.Interface) (*Controller, error) {
	logger := logr.FromContextOrDiscard(ctx)

	limiter := workqueue.DefaultControllerRateLimiter()

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logger.Info)
	eventBroadcaster.StartRecordingToSink(&v12.EventSinkImpl{Interface: kubeClient.CoreV1().Events(v1.NamespaceAll)})

	daemonSetHelper := &appsV1.DaemonSet{}
	manifest := utils.FileOrContent(opt.DaemonSetManifest)

	if err := json.Unmarshal([]byte(manifest), daemonSetHelper); err != nil {
		logger.Error(err, "unable Unmarshal manifest: %s into struct, got: %v", manifest, err)
		return nil, err
	}

	ctl := &Controller{
		options:            opt,
		kubeClient:         kubeClient,
		daemonSetHelper:    daemonSetHelper,
		serviceLister:      serviceInformer.Lister(),
		daemonSetLister:    daemonSetInformer.Lister(),
		serviceHasSynced:   serviceInformer.Informer().HasSynced,
		daemonSetHasSynced: daemonSetInformer.Informer().HasSynced,
		serviceQueue:       workqueue.NewRateLimitingQueue(limiter),
		daemonSetQueue:     workqueue.NewRateLimitingQueue(limiter),
		eventRecord:        eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerId}),
	}

	serviceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.addService,
			UpdateFunc: ctl.updateService,
			DeleteFunc: ctl.deleteService,
		},
	)

	daemonSetInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.addDaemonSet,
			UpdateFunc: ctl.updateDaemonSet,
			DeleteFunc: ctl.deleteDaemonSet,
		},
	)
	return ctl, nil
}
