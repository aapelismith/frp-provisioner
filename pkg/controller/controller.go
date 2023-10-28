/*
 * Copyright 2021 Aapeli.Smith<aapeli.nian@gmail.com>.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"context"
	"fmt"
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"github.com/aapelismith/frp-provisioner/pkg/safe"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sync"
	"time"
)

// maxRetries is the number of times an endpoint will be retried before it is dropped out of the queue.
// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
// an endpoint is going to be requeued:
//
// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
const (
	maxRetries     = 10
	controllerName = "frp-provisioner"
)

var _ manager.Runnable = (*Controller)(nil)

type Controller struct {
	safe.NoCopy
	logger     logr.Logger
	options    *config.FrpOptions
	client     kubernetes.Interface
	handlers   sync.Map
	nodeLister listers.NodeLister
	lister     listers.ServiceLister
	hasSynced  []cache.InformerSynced
	queue      workqueue.RateLimitingInterface
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error(err, "unable get namespace key from object")
		return
	}
	c.queue.Add(key)
}

func (c *Controller) Start(ctx context.Context) error {
	c.logger.Info("Starting controller, please wait...")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.hasSynced...); !ok {
		return fmt.Errorf("failed to wait for caches to sync for: %s", controllerName)
	}

	safe.Go(func() {
		defer c.queue.ShutDown()
		<-ctx.Done()
	})

	wait.UntilWithContext(ctx, c.worker, time.Second)
	return nil
}

// SetupWithManager init current controller with manager.Manager
func (c *Controller) SetupWithManager(mgr manager.Manager) error {
	c.logger = mgr.GetLogger().WithValues(
		"controller", controllerName,
	)
	if err := mgr.Add(c); err != nil {
		c.logger.Error(err, "add controller to manager failed")
		return err
	}
	return nil
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Controller) worker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	if err := ctx.Err(); err != nil {
		return false
	}

	key, quit := c.queue.Get()
	if quit {
		c.logger.Info("Ignore work item and stop working.")
		return false
	}
	defer c.queue.Done(key)

	c.handleErr(c.syncHandler(ctx, key.(string)), key)
	return true
}

func (c *Controller) syncHandler(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.logger.Error(err, "Unable split namespace and name for key: "+key)
		return err
	}

	svc, err := c.lister.Services(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		c.logger.Error(err, "Unable get service by name", "namespace", namespace, "name", name)
		return err
	}

	_ = svc

	if errors.IsNotFound(err) {
		// TODO: shutdown frp service
	}

	if err == nil {
		// TODO: check and start
	}
	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < maxRetries {
		c.queue.AddRateLimited(key)
		return
	}

	c.logger.Error(err, "Dropping service out of queue", "name", key)
	c.queue.Forget(key)
}

func NewController(
	baseCtx context.Context,
	options *config.FrpOptions,
	client kubernetes.Interface,
	serviceInformer informers.ServiceInformer,
	nodeInformer informers.NodeInformer,
) (*Controller, error) {
	ctx := log.NewContext(baseCtx, log.FromContext(baseCtx).
		With(zap.String("controller", controllerName)))
	logger := log.FromContext(ctx).Sugar()

	rateLimiter := workqueue.DefaultControllerRateLimiter()

	ctrl := &Controller{
		options: options,
		client:  client,
		hasSynced: []cache.InformerSynced{
			nodeInformer.Informer().HasSynced,
			serviceInformer.Informer().HasSynced,
		},
		nodeLister: nodeInformer.Lister(),
		lister:     serviceInformer.Lister(),
		queue:      workqueue.NewRateLimitingQueue(rateLimiter),
	}

	handlerFunc := cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			ctrl.enqueue(newObj)
		},
		DeleteFunc: ctrl.enqueue,
	}

	_, err := serviceInformer.Informer().AddEventHandler(handlerFunc)
	if err != nil {
		logger.With(zap.Error(err)).Errorln("unable add event handler for service informer")
		return nil, fmt.Errorf("unable add event handler for service informer: %w", err)
	}
	return ctrl, nil
}
