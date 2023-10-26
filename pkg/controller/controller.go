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
	"go.uber.org/zap"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ manager.Runnable = (*Controller)(nil)

type Controller struct {
	safe.NoCopy
	options       *config.FrpOptions
	client        kubernetes.Interface
	nodeLister    listers.NodeLister
	serviceLister listers.ServiceLister
	hasSynced     []cache.InformerSynced
	nodeQueue     workqueue.RateLimitingInterface
	serviceQueue  workqueue.RateLimitingInterface
}

func (c *Controller) enqueueNode(obj interface{}) {

}

func (c *Controller) enqueueService(obj interface{}) {
}

func (c *Controller) Start(ctx context.Context) error {
	return nil
}

func (c *Controller) SetupWithManager(mgr manager.Manager) error {
	return nil
}

func NewController(
	ctx context.Context,
	options *config.FrpOptions,
	client kubernetes.Interface,
	serviceInformer informers.ServiceInformer,
	nodeInformer informers.NodeInformer,
) (*Controller, error) {
	logger := log.FromContext(ctx).Sugar()
	nodeRateLimiter := workqueue.DefaultControllerRateLimiter()
	serviceRateLimiter := workqueue.DefaultControllerRateLimiter()

	ctrl := &Controller{
		options: options,
		client:  client,
		hasSynced: []cache.InformerSynced{
			nodeInformer.Informer().HasSynced,
			serviceInformer.Informer().HasSynced,
		},
		nodeLister:    nodeInformer.Lister(),
		serviceLister: serviceInformer.Lister(),
		nodeQueue:     workqueue.NewRateLimitingQueue(nodeRateLimiter),
		serviceQueue:  workqueue.NewRateLimitingQueue(serviceRateLimiter),
	}

	nodeHandlerFunc := cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enqueueNode,
		UpdateFunc: func(_, newObj interface{}) {
			ctrl.enqueueNode(newObj)
		},
		DeleteFunc: ctrl.enqueueNode,
	}

	serviceHandlerFunc := cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enqueueService,
		UpdateFunc: func(_, newObj interface{}) {
			ctrl.enqueueService(newObj)
		},
		DeleteFunc: ctrl.enqueueService,
	}

	_, err := nodeInformer.Informer().AddEventHandler(nodeHandlerFunc)
	if err != nil {
		logger.With(zap.Error(err)).Errorln("unable add event handler for node informer")
		return nil, fmt.Errorf("unable add event handler for node informer: %w", err)
	}

	_, err = serviceInformer.Informer().AddEventHandler(serviceHandlerFunc)
	if err != nil {
		logger.With(zap.Error(err)).Errorln("unable add event handler for service informer")
		return nil, fmt.Errorf("unable add event handler for service informer: %w", err)
	}

	return ctrl, nil
}
