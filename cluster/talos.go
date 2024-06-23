package cluster

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/client"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
)

func CreateClusterClientCfg(ctx *pulumi.Context, clusterName string, publicIp pulumi.StringPtrInput) (*client.GetConfigurationResultOutput, error) {
	thisSecrets, err := machine.NewSecrets(ctx, "machineSecret", nil)
	if err != nil {
		return nil, err
	}
	res := client.GetConfigurationOutput(ctx, client.GetConfigurationOutputArgs{
		ClusterName: pulumi.String(clusterName),
		ClientConfiguration: client.GetConfigurationClientConfigurationArgs{
			CaCertificate:     thisSecrets.ClientConfiguration.CaCertificate(),
			ClientCertificate: thisSecrets.ClientConfiguration.ClientCertificate(),
			ClientKey:         thisSecrets.ClientConfiguration.ClientKey(),
		},
		Nodes: pulumi.StringArray{
			publicIp.ToStringPtrOutput().Elem().ToStringOutput(),
		},
	})
	return &res, err
}
