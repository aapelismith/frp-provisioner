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

package controller

import (
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

func IsPodActive(p *v1.Pod) bool {
	return v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase &&
		p.DeletionTimestamp == nil
}

func IsFrpServerActive(i *v1beta1.FrpServer) bool {
	return i.Status.Phase == v1beta1.FrpServerPhaseHealthy &&
		i.DeletionTimestamp == nil
}
