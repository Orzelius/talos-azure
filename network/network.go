package network

import (
	"fmt"
	"talos-azure/helpers"

	"github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NetworkResources struct {
	Vnet                      *network.VirtualNetwork
	NetworkSecurityGroup      *network.NetworkSecurityGroup
	PublicLbIp                *network.PublicIPAddress
	PublicNatIp               *network.PublicIPAddress
	LoadBalancer              *network.LoadBalancer
	InboundNatRule            *network.InboundNatRule
	NetworkInterfaces         []*network.NetworkInterface
	NetworkInterfacePublicIPs []*network.PublicIPAddress
	NatGateway                *network.NatGateway
}

type ProvisionNetworkingParams struct {
	ResourceGroup         *resources.ResourceGroup
	ControlplaneNodeCount int
}

func ProvisionNetworking(ctx *pulumi.Context, params ProvisionNetworkingParams) (NetworkResources, error) {
	conf, err := helpers.GetConfig(ctx)
	if err != nil {
		return NetworkResources{}, err
	}

	publicNatIp, err := network.NewPublicIPAddress(ctx, "public-nat-ip", &network.PublicIPAddressArgs{
		PublicIPAllocationMethod: pulumi.String("static"),
		ResourceGroupName:        params.ResourceGroup.Name,
		Sku: network.PublicIPAddressSkuArgs{
			Name: pulumi.String(network.PublicIPAddressSkuNameStandard),
		},
	})
	if err != nil {
		return NetworkResources{}, err
	}

	natGateway, err := network.NewNatGateway(ctx, "natGateway", &network.NatGatewayArgs{
		PublicIpAddresses: network.SubResourceArray{
			&network.SubResourceArgs{
				Id: publicNatIp.ID(),
			},
		},
		ResourceGroupName: params.ResourceGroup.Name,
		Sku: &network.NatGatewaySkuArgs{
			Name: pulumi.String(network.NatGatewaySkuNameStandard),
		},
	})
	if err != nil {
		return NetworkResources{}, err
	}

	var subnet = network.SubnetTypeArgs{
		Name:          pulumi.String("subnet"),
		AddressPrefix: pulumi.String("10.0.0.0/24"),
		NatGateway: network.SubResourceArgs{
			Id: natGateway.ID(),
		},
	}
	subnet = *subnet.Defaults()
	vnet, err := network.NewVirtualNetwork(ctx, "vnet", &network.VirtualNetworkArgs{
		AddressSpace: &network.AddressSpaceArgs{
			AddressPrefixes: pulumi.StringArray{
				pulumi.String("10.0.0.0/16"),
			},
		},
		FlowTimeoutInMinutes: pulumi.Int(10),
		Location:             pulumi.String(conf.AzRegion),
		ResourceGroupName:    params.ResourceGroup.Name,
		VirtualNetworkName:   pulumi.String("vnet"),
		Subnets:              network.SubnetTypeArray{subnet},
	})
	if err != nil {
		return NetworkResources{}, err
	}

	networkSecurityGroup, err := network.NewNetworkSecurityGroup(ctx, "nsg",
		&network.NetworkSecurityGroupArgs{
			ResourceGroupName: params.ResourceGroup.Name,
			SecurityRules: network.SecurityRuleTypeArray{
				makeSecurityRule(securityRuleParams{name: "apid", DestinationPortRange: "50000"}),
				makeSecurityRule(securityRuleParams{name: "trustd", DestinationPortRange: "50001"}),
				makeSecurityRule(securityRuleParams{name: "etcd", DestinationPortRange: "2379-2380"}),
				makeSecurityRule(securityRuleParams{name: "kube", DestinationPortRange: "6443"}),
			}},
	)
	if err != nil {
		return NetworkResources{}, err
	}

	publicLbIp, err := network.NewPublicIPAddress(ctx, "public-lb-ip", &network.PublicIPAddressArgs{
		PublicIPAllocationMethod: pulumi.String("static"),
		ResourceGroupName:        params.ResourceGroup.Name,
		Sku: network.PublicIPAddressSkuArgs{
			Name: pulumi.String(network.PublicIPAddressSkuNameStandard),
		},
	})
	if err != nil {
		return NetworkResources{}, err
	}

	lb, err := network.NewLoadBalancer(ctx, "lb", &network.LoadBalancerArgs{
		FrontendIPConfigurations: network.FrontendIPConfigurationArray{
			network.FrontendIPConfigurationArgs{
				Name: pulumi.String("talos-fe"),
				PublicIPAddress: network.PublicIPAddressTypeArgs{
					IpAddress: publicLbIp.IpAddress,
					Id:        publicLbIp.ID(),
				},
			},
		},
		Sku: network.LoadBalancerSkuArgs{
			Name: pulumi.String(network.LoadBalancerSkuNameStandard),
		},
		BackendAddressPools: network.BackendAddressPoolArray{network.BackendAddressPoolArgs{
			Name: pulumi.String("talos-be-pool"),
		}},
		ResourceGroupName: params.ResourceGroup.Name,
		Probes: network.ProbeArray{network.ProbeArgs{
			Name:     pulumi.String("talos-lb-health"),
			Port:     pulumi.Int(6443),
			Protocol: pulumi.String("TCP"),
		}},
	})
	if err != nil {
		return NetworkResources{}, err
	}

	lbRule, err := network.NewInboundNatRule(ctx, "talos-6443", &network.InboundNatRuleArgs{
		ResourceGroupName: params.ResourceGroup.Name,
		Protocol:          pulumi.String("TCP"),
		FrontendIPConfiguration: network.SubResourceArgs{
			Id: lb.FrontendIPConfigurations.Index(pulumi.Int(0)).Id(),
		},
		BackendAddressPool: network.SubResourceArgs{
			Id: lb.BackendAddressPools.Index(pulumi.Int(0)).Id(),
		},
		FrontendPortRangeStart: pulumi.Int(6443),
		FrontendPortRangeEnd:   pulumi.Int(6443),
		BackendPort:            pulumi.Int(6443),
		LoadBalancerName:       lb.Name,
	})
	if err != nil {
		return NetworkResources{}, err
	}

	nicPubIps := make([]*network.PublicIPAddress, params.ControlplaneNodeCount)
	nics := make([]*network.NetworkInterface, params.ControlplaneNodeCount)
	for i := 0; i < params.ControlplaneNodeCount; i++ {
		nicPubIp, err := network.NewPublicIPAddress(ctx, fmt.Sprintf("controlplane-public-ip-%d", i),
			&network.PublicIPAddressArgs{
				ResourceGroupName:        params.ResourceGroup.Name,
				PublicIPAllocationMethod: pulumi.String("static"),
				Sku: network.PublicIPAddressSkuArgs{
					Name: pulumi.String(network.PublicIPAddressSkuNameStandard),
				},
			})
		if err != nil {
			return NetworkResources{}, err
		}
		nicPubIps[i] = nicPubIp

		nicName := fmt.Sprintf("controlplane-nic-%d", i)
		nic, err := network.NewNetworkInterface(ctx, nicName,
			&network.NetworkInterfaceArgs{
				ResourceGroupName:    params.ResourceGroup.Name,
				NetworkInterfaceName: pulumi.String(nicName),
				NetworkSecurityGroup: network.NetworkSecurityGroupTypeArgs{
					Id: networkSecurityGroup.ID(),
				},
				IpConfigurations: network.NetworkInterfaceIPConfigurationArray{network.NetworkInterfaceIPConfigurationArgs{
					Name:            pulumi.String(fmt.Sprintf("%s-ip-conf", nicName)),
					PublicIPAddress: network.PublicIPAddressTypeArgs{Id: nicPubIp.ID()},
					Subnet:          network.SubnetTypeArgs{Id: vnet.Subnets.Index(pulumi.Int(0)).Id()},
				}},
			})
		if err != nil {
			return NetworkResources{}, err
		}
		nics[i] = nic
	}

	return NetworkResources{vnet, networkSecurityGroup, publicLbIp, publicNatIp, lb, lbRule, nics, nicPubIps, natGateway}, nil
}
