/*
 * Copyright 2021 The KunStack Authors.
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

package v1beta1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmDeploy is a specification for a Helm Release resource
type HelmDeploy struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          HelmDeploySpec   `json:"spec"`
	Status        HelmDeployStatus `json:"status"`
}

// HelmDeploySpec is the spec for a Helm resource
type HelmDeploySpec struct {
	// Wait if set, will wait until all Pods, PVCs, Services,
	// and minimum number of Pods of a Deployment, StatefulSet, or
	// ReplicaSet are in a ready state before marking the release as successful.
	Wait bool `json:"wait,omitempty"`
	//Force if set, will delete/recreate resource when necessary
	Force bool `json:"force,omitempty"`
	// Values helm release values.yaml content
	Values string `json:"values,omitempty"`
	// Timeout to wait for installation/update to complete
	Timeout int64 `json:"timeout,omitempty"`
	// Description Message for helm description
	Description string `json:"description,omitempty"`
	// DisableHooks disable helm pre/post upgrade hooks
	DisableHooks bool `json:"disableHooks,omitempty"`
}

// HelmDeployStatus is the status for a Helm resource
type HelmDeployStatus struct {
	// State current state of the release
	State string `json:"state,omitempty"`
	// Version current release version
	Version int64 `json:"version,omitempty"`
	// Message human-readable message indicating details about why the release is in this state.
	Message string `json:"message,omitempty"`
	// Manifest helm release manifest
	Manifest string `json:"manifest,omitempty"`
	// LastDeployed last deploy time
	LastDeployed *v1.Time `json:"lastDeployed,omitempty"`
	// LastUpdate last update time
	LastUpdate *v1.Time `json:"lastUpdate,omitempty"`
	// DeployStatus deploy status list of history, which will store at most 10 state
	DeployStatus []DeployStatus `json:"deployStatus,omitempty"`
}

// DeployStatus deploy status of history
type DeployStatus struct {
	// State current release state
	State string `json:"state,omitempty"`
	// Version current release version
	Version int64 `json:"version,omitempty"`
	// Message human-readable message indicating details about why the release is in this state.
	Message string `json:"message,omitempty"`
	// DeployTime last deploy time
	DeployTime *v1.Time `json:"deployTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmDeployList is a list of helmDeploy resources
type HelmDeployList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	// Items helm deploy items
	Items []HelmDeploy `json:"items,omitempty"`
}
