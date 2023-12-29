/*
Copyright 2023 Aapeli <aapeli.smith@gmail.com>.

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

package fieldindex

import (
	"context"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	IndexNameForOwnerRefUID    = "ownerRefUID"
	IndexNameForFrpServerPhase = "status.phase"
)

var ownerIndexFunc = func(obj client.Object) []string {
	var owners []string
	for _, ref := range obj.GetOwnerReferences() {
		owners = append(owners, string(ref.UID))
	}
	return owners
}

var phaseIndexFunc = func(obj client.Object) []string {
	srv, ok := obj.(*v1beta1.FrpServer)
	if !ok {
		return []string{}
	}
	if len(srv.Status.Phase) == 0 {
		return []string{}
	}
	return []string{string(srv.Status.Phase)}
}

func RegisterFieldIndexes(ctx context.Context, c cache.Cache) error {
	logger := log.FromContext(ctx)
	// pod ownerReference
	if err := c.IndexField(ctx, &v1.Pod{}, IndexNameForOwnerRefUID, ownerIndexFunc); err != nil {
		logger.Error(err, "unable register index filed for pod")
		return err
	}

	if err := c.IndexField(ctx, &v1beta1.FrpServer{}, IndexNameForFrpServerPhase, phaseIndexFunc); err != nil {
		logger.Error(err, "unable register index filed for FrpServer")
		return err
	}
	return nil
}
