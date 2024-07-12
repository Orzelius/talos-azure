package cluster

import (
	"encoding/base64"
	"fmt"
	"talos-azure/helpers"

	compute "github.com/pulumi/pulumi-azure-native-sdk/compute/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ComputeResources struct {
	AvailabilitySet *compute.AvailabilitySet
	Nodes           []*compute.VirtualMachine
}

type ProvisionComputeParams struct {
	ResourceGroup  *resources.ResourceGroup
	MachineConfigs MachineConfigs
	NicIds         []pulumi.IDOutput
	StorageAccUri  pulumi.StringPtrInput
	SubnetID       pulumi.StringPtrOutput
	NsgId          pulumi.IDOutput
}

func ProvisionCompute(ctx *pulumi.Context, params ProvisionComputeParams) (ComputeResources, error) {
	conf, err := helpers.GetConfig(ctx)
	if err != nil {
		return ComputeResources{}, err
	}

	availabilitySet, err := compute.NewAvailabilitySet(ctx, "availabilitySet", &compute.AvailabilitySetArgs{
		ResourceGroupName: params.ResourceGroup.Name,
		Location:          pulumi.String(conf.AzRegion),
		Sku: compute.SkuArgs{
			Name: pulumi.StringPtr("Aligned"),
		},
		PlatformFaultDomainCount: pulumi.Int(2),
	})
	if err != nil {
		return ComputeResources{}, err
	}

	imageId := pulumi.Sprintf(
		"/CommunityGalleries/siderolabs-c4d707c0-343e-42de-b597-276e4f7a5b0b/Images/%s/Versions/%s",
		conf.Architecture,
		conf.TalosVersion,
	)

	nodes := make([]*compute.VirtualMachine, 0)
	for i := 0; i < conf.ControlCount; i++ {
		name := fmt.Sprintf("control-%d", i)
		node, err := createNode(ctx, params, createNodeParams{
			name:              name,
			imageId:           imageId,
			availabilitySetID: availabilitySet.ID(),
			isControlplane:    true,
			nicID:             params.NicIds[i],
			subnetID:          params.SubnetID,
			nsgId:             params.NsgId,
		})
		if err != nil {
			return ComputeResources{}, err
		}
		nodes = append(nodes, node)
	}
	for i := 0; i < conf.WorkerCount; i++ {
		name := fmt.Sprintf("worker-%d", i)
		node, err := createNode(ctx, params, createNodeParams{
			name:              name,
			imageId:           imageId,
			availabilitySetID: availabilitySet.ID(),
			isControlplane:    false,
			nicID:             pulumi.IDOutput{},
			subnetID:          params.SubnetID,
			nsgId:             params.NsgId,
		})
		if err != nil {
			return ComputeResources{}, err
		}
		nodes = append(nodes, node)
	}

	return ComputeResources{availabilitySet, nodes}, nil
}

type createNodeParams struct {
	name              string
	imageId           pulumi.StringOutput
	availabilitySetID pulumi.StringPtrInput
	isControlplane    bool
	nicID             pulumi.IDOutput
	subnetID          pulumi.StringPtrInput
	nsgId             pulumi.IDOutput
}

func createNode(ctx *pulumi.Context, params ProvisionComputeParams, nodeParams createNodeParams) (*compute.VirtualMachine, error) {
	var machineCfg pulumi.StringOutput
	var networkInterfaces compute.NetworkInterfaceReferenceArray
	var networkInterfaceConfiguration compute.VirtualMachineNetworkInterfaceConfigurationArray
	var networkApiVersion pulumi.StringPtrInput
	if nodeParams.isControlplane {
		networkInterfaces = compute.NetworkInterfaceReferenceArray{&compute.NetworkInterfaceReferenceArgs{
			Id: nodeParams.nicID,
		}}
		machineCfg = params.MachineConfigs.Controlplane.MachineConfiguration()
	} else {
		networkApiVersion = compute.NetworkApiVersion("2023-11-01")
		machineCfg = params.MachineConfigs.Worker.MachineConfiguration()
		networkInterfaceConfiguration = compute.VirtualMachineNetworkInterfaceConfigurationArray{&compute.VirtualMachineNetworkInterfaceConfigurationArgs{
			Name: pulumi.String(nodeParams.name),
			NetworkSecurityGroup: compute.SubResourceArgs{
				Id: nodeParams.nsgId,
			},
			IpConfigurations: compute.VirtualMachineNetworkInterfaceIPConfigurationArray{&compute.VirtualMachineNetworkInterfaceIPConfigurationArgs{
				Name: pulumi.String(nodeParams.name),
				Subnet: compute.SubResourceArgs{
					Id: nodeParams.subnetID,
				},
			}},
		}}
	}

	var networkProfile = compute.NetworkProfileArgs{
		// https://learn.microsoft.com/en-us/azure/templates/microsoft.network/change-log/virtualnetworks
		NetworkApiVersion:              networkApiVersion,
		NetworkInterfaces:              networkInterfaces,
		NetworkInterfaceConfigurations: networkInterfaceConfiguration,
	}

	return compute.NewVirtualMachine(ctx, nodeParams.name, &compute.VirtualMachineArgs{
		ResourceGroupName: params.ResourceGroup.Name,
		HardwareProfile: &compute.HardwareProfileArgs{
			VmSize: pulumi.String(compute.VirtualMachineSizeTypes_Standard_B1s),
		},
		StorageProfile: compute.StorageProfileArgs{
			ImageReference: &compute.ImageReferenceArgs{
				CommunityGalleryImageId: nodeParams.imageId,
			},
			OsDisk: compute.OSDiskArgs{
				DiskSizeGB:   pulumi.Int(10),
				CreateOption: pulumi.String(compute.DiskCreateOptionTypesFromImage)},
		},
		OsProfile: compute.OSProfileArgs{
			CustomData:   machineCfg.ApplyT(func(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) }).(pulumi.StringOutput),
			ComputerName: pulumi.String(nodeParams.name),
			// The following two are not used, but are required by the api
			AdminUsername: pulumi.String("talos"),
			AdminPassword: pulumi.String("talosASD123&€#"),
		},
		DiagnosticsProfile: &compute.DiagnosticsProfileArgs{
			BootDiagnostics: &compute.BootDiagnosticsArgs{
				Enabled:    pulumi.Bool(true),
				StorageUri: params.StorageAccUri,
			},
		},
		NetworkProfile: &networkProfile,
		AvailabilitySet: compute.SubResourceArgs{
			Id: nodeParams.availabilitySetID,
		},
	})
}
