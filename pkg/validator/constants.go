package validator

import "github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"

const defaultNatHoleSTUNServer = "stun.easyvoip.com:3478"

var (
	authMethods = []v1beta1.FrpServerAuthMethod{
		v1beta1.FrpServerAuthMethodToken,
		v1beta1.FrpServerAuthMethodOIDC,
	}
	authScopes = []v1beta1.FrpServerAuthScope{
		v1beta1.FrpServerAuthScopeHeartBeats,
		v1beta1.FrpServerAuthScopeNewWorkConns,
	}
	transportProtocols = []v1beta1.FrpServerTransportProtocol{
		v1beta1.FrpServerTransportProtocolTCP,
		v1beta1.FrpServerTransportProtocolKCP,
		v1beta1.FrpServerTransportProtocolQUIC,
		v1beta1.FrpServerTransportProtocolWSS,
		v1beta1.FrpServerTransportProtocolWebsocket,
	}
)
