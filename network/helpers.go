package network

import (
	"github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type securityRuleParams struct {
	name                 string
	DestinationPortRange string
}

var rulePrio = 1000

func makeSecurityRule(params securityRuleParams) network.SecurityRuleTypeArgs {
	rulePrio++
	return network.SecurityRuleTypeArgs{
		Name:                     pulumi.String(params.name),
		DestinationPortRange:     pulumi.String(params.DestinationPortRange),
		Direction:                pulumi.String("inbound"),
		Protocol:                 pulumi.String("TCP"),
		Access:                   pulumi.String("Allow"),
		SourcePortRange:          pulumi.String("*"),
		SourceAddressPrefix:      pulumi.String("*"),
		DestinationAddressPrefix: pulumi.String("*"),
		Priority:                 pulumi.IntPtr(rulePrio),
	}
}
