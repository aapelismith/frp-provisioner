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

package v1beta1

var (
	FrpServerAuthMethods = []FrpServerAuthMethod{
		FrpServerAuthMethodToken,
		FrpServerAuthMethodOIDC,
	}
	FrpServerAuthScopes = []FrpServerAuthScope{
		FrpServerAuthScopeHeartBeats,
		FrpServerAuthScopeNewWorkConns,
	}
	FrpServerTransportProtocols = []FrpServerTransportProtocol{
		FrpServerTransportProtocolTCP,
		FrpServerTransportProtocolKCP,
		FrpServerTransportProtocolQUIC,
		FrpServerTransportProtocolWSS,
		FrpServerTransportProtocolWebsocket,
	}
)

const (
	FinalizerName              string = "finalizer.gofrp.io/tracking"
	LabelServiceNameKey        string = "gofrp.io/service-name"
	AnnotationFrpServerNameKey string = "service.beta.kubernetes.io/frp-server-name"

	DefaultCaFileName      = "tls.ca"
	DefaultCertFileName    = "tls.crt"
	DefaultKeyFileName     = "tls.key"
	DefaultNatHoleSTUNAddr = "stun.easyvoip.com:3478"
)
