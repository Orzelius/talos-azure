package cluster

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/client"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
)

func GetMachineSecrets(ctx *pulumi.Context, clusterName string, publicIp pulumi.StringPtrInput) (*machine.Secrets, error) {
	thisSecrets, err := machine.NewSecrets(ctx, "machineSecret", nil)
	if err != nil {
		return nil, err
	}
	return thisSecrets, err
}

type CommonProps struct {
	ClusterName string
	PublicIp    pulumi.StringPtrInput
	Secrets     *machine.Secrets
}

func GetClusterClientCfg(ctx *pulumi.Context, props CommonProps) *client.GetConfigurationResultOutput {
	res := client.GetConfigurationOutput(ctx, client.GetConfigurationOutputArgs{
		ClusterName: pulumi.String(props.ClusterName),
		ClientConfiguration: client.GetConfigurationClientConfigurationArgs{
			CaCertificate:     props.Secrets.ClientConfiguration.CaCertificate(),
			ClientCertificate: props.Secrets.ClientConfiguration.ClientCertificate(),
			ClientKey:         props.Secrets.ClientConfiguration.ClientKey(),
		},
		Nodes: pulumi.StringArray{
			props.PublicIp.ToStringPtrOutput().Elem().ToStringOutput(),
		},
	})
	return &res
}

type MachineConfigs struct {
	Controlplane *machine.GetConfigurationResultOutput
	Worker       *machine.GetConfigurationResultOutput
}

func GetMachineConfiguration(ctx *pulumi.Context, props CommonProps) MachineConfigs {
	enpoint := props.PublicIp.ToStringPtrOutput().ApplyT(func(ip *string) string {
		return fmt.Sprintf("https://%s:6443", *ip)
	}).(pulumi.StringOutput)
	control := machine.GetConfigurationOutput(ctx, machine.GetConfigurationOutputArgs{
		ClusterName:     pulumi.String(props.ClusterName),
		MachineSecrets:  props.Secrets.MachineSecrets,
		ClusterEndpoint: enpoint,
		MachineType:     pulumi.String("controlplane"),
	},
	)
	worker := machine.GetConfigurationOutput(ctx, machine.GetConfigurationOutputArgs{
		ClusterName:     pulumi.String(props.ClusterName),
		MachineSecrets:  props.Secrets.MachineSecrets,
		ClusterEndpoint: enpoint,
		MachineType:     pulumi.String("worker"),
	},
	)
	return MachineConfigs{
		Controlplane: &control,
		Worker:       &worker,
	}
}
