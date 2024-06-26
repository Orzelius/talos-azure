package cluster

import (
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

	node, err := compute.NewVirtualMachine(ctx, "node", &compute.VirtualMachineArgs{
		ResourceGroupName: params.ResourceGroup.Name,
		HardwareProfile: &compute.HardwareProfileArgs{
			VmSize: pulumi.String(compute.VirtualMachineSizeTypes_Standard_B1s),
		},
		StorageProfile: compute.StorageProfileArgs{
			ImageReference: &compute.ImageReferenceArgs{
				CommunityGalleryImageId: imageId,
			},
			OsDisk: compute.OSDiskArgs{
				DiskSizeGB:   pulumi.Int(10),
				CreateOption: pulumi.String(compute.DiskCreateOptionTypesFromImage)},
		},
		OsProfile: compute.OSProfileArgs{
			CustomData: params.MachineConfigs.Controlplane.MachineConfiguration(),
			// AdminUsername: pulumi.String("talos"),
		},
		DiagnosticsProfile: &compute.DiagnosticsProfileArgs{
			BootDiagnostics: &compute.BootDiagnosticsArgs{
				Enabled:    pulumi.Bool(true),
				StorageUri: params.StorageAccUri,
			},
		},
		NetworkProfile: compute.NetworkProfileArgs{
			NetworkInterfaces: compute.NetworkInterfaceReferenceArray{compute.NetworkInterfaceReferenceArgs{
				Id: params.NicIds[0],
			}},
		},
		AvailabilitySet: compute.SubResourceArgs{
			Id: availabilitySet.ID(),
		},
	})
	if err != nil {
		return ComputeResources{}, err
	}

	return ComputeResources{availabilitySet, []*compute.VirtualMachine{node}}, nil
}
