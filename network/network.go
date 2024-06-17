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
	PublicIp                  *network.PublicIPAddress
	LoadBalancer              *network.LoadBalancer
	InboundNatRule            *network.InboundNatRule
	networkInterfaces         []*network.NetworkInterface
	networkInterfacePublicIPs []*network.PublicIPAddress
}

type ProvisionNetworkingParams struct {
	ResourceGroup         *resources.ResourceGroup
	ControlPlaneNodeCount int
}

func ProvisionNetworking(ctx *pulumi.Context, params ProvisionNetworkingParams) (NetworkResources, error) {
	conf, err := helpers.GetConfig(ctx)
	fmt.Println(conf.AzRegion)
	if err != nil {
		return NetworkResources{}, err
	}

	var subnet = network.SubnetTypeArgs{
		Name:          pulumi.String("subnet"),
		AddressPrefix: pulumi.String("10.0.0.0/24"),
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

	publicIp, err := network.NewPublicIPAddress(ctx, "public-ip", &network.PublicIPAddressArgs{
		PublicIPAllocationMethod: pulumi.String("static"),
		ResourceGroupName:        params.ResourceGroup.Name,
	})
	if err != nil {
		return NetworkResources{}, err
	}

	lb, err := network.NewLoadBalancer(ctx, "lb", &network.LoadBalancerArgs{
		FrontendIPConfigurations: network.FrontendIPConfigurationArray{
			network.FrontendIPConfigurationArgs{
				Name: pulumi.String("talos-fe"),
				PublicIPAddress: network.PublicIPAddressTypeArgs{
					IpAddress: publicIp.IpAddress,
					Id:        publicIp.ID(),
				},
			},
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

	nicPubIps := make([]*network.PublicIPAddress, params.ControlPlaneNodeCount)
	nics := make([]*network.NetworkInterface, params.ControlPlaneNodeCount)
	for i := 0; i < params.ControlPlaneNodeCount; i++ {
		nicPubIp, err := network.NewPublicIPAddress(ctx, fmt.Sprintf("controlplane-public-ip-%d", i),
			&network.PublicIPAddressArgs{
				ResourceGroupName:        params.ResourceGroup.Name,
				PublicIPAllocationMethod: pulumi.String("static"),
			})
		if err != nil {
			return NetworkResources{}, err
		}
		nicPubIps = append(nicPubIps, nicPubIp)

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
		nics = append(nics, nic)
	}

	return NetworkResources{vnet, networkSecurityGroup, publicIp, lb, lbRule, nics, nicPubIps}, nil
}
