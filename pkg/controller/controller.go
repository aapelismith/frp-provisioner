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
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/safe"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ manager.Runnable = (*Controller)(nil)

type Controller struct {
	safe.NoCopy
	Client client.Client
	Scheme *runtime.Scheme
}

func (c *Controller) Start(ctx context.Context) error {
	return nil
}

func (c *Controller) SetupWithManager(mgr manager.Manager) error {
	return nil
}

func NewController(options *config.FrpOptions) (*Controller, error) {
	return nil, nil
}
