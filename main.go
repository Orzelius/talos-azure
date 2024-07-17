package main

import (
	"fmt"
	"os"
	"talos-azure/cluster"
	"talos-azure/helpers"
	"talos-azure/network"

	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/storage/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf, err := helpers.GetConfig(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("azure region: %s", conf.AzRegion)
		// Create an Azure Resource Group
		resourceGroup, err := resources.NewResourceGroup(ctx, conf.ResourceGroupName, nil)
		if err != nil {
			return err
		}

		// Create an Azure resource (Storage Account)
		storageAcc, err := storage.NewStorageAccount(ctx, "sa", &storage.StorageAccountArgs{
			ResourceGroupName: resourceGroup.Name,
			Sku: &storage.SkuArgs{
				Name: pulumi.String("Standard_LRS"),
			},
			Kind: pulumi.String("StorageV2"),
		})
		if err != nil {
			return err
		}

		// Export the primary key of the Storage Account
		ctx.Export("primaryStorageKey", pulumi.All(resourceGroup.Name, storageAcc.Name).ApplyT(
			func(args []interface{}) (string, error) {
				resourceGroupName := args[0].(string)
				accountName := args[1].(string)
				accountKeys, err := storage.ListStorageAccountKeys(ctx, &storage.ListStorageAccountKeysArgs{
					ResourceGroupName: resourceGroupName,
					AccountName:       accountName,
				})
				if err != nil {
					return "", err
				}

				return accountKeys.Keys[0].Value, nil
			},
		))

		networkResources, err := network.ProvisionNetworking(ctx, network.ProvisionNetworkingParams{
			ResourceGroup: resourceGroup,
		})
		if err != nil {
			return err
		}

		clusterSecrets, err := cluster.GetMachineSecrets(ctx, conf.ClusterName, networkResources.PublicLbIp.IpAddress)
		if err != nil {
			return err
		}

		commonTalosProps := cluster.CommonProps{
			ClusterName: conf.ClusterName,
			PublicIp:    networkResources.PublicLbIp.IpAddress,
			Secrets:     clusterSecrets,
		}
		clusterClientCfg := cluster.GetClusterClientCfg(ctx, commonTalosProps)

		machineCfg := cluster.GetMachineConfiguration(ctx, commonTalosProps)

		controlNicIds := make([]pulumi.IDOutput, len(networkResources.ControlNetworkInterfaces))
		for i, nic := range networkResources.ControlNetworkInterfaces {
			controlNicIds[i] = nic.ID()
		}

		workerNicIds := make([]pulumi.IDOutput, len(networkResources.WorkerNetworkInterfaces))
		for i, nic := range networkResources.WorkerNetworkInterfaces {
			workerNicIds[i] = nic.ID()
		}
		_, err = cluster.ProvisionCompute(ctx, cluster.ProvisionComputeParams{
			ResourceGroup:  resourceGroup,
			MachineConfigs: machineCfg,
			WorkerNicIds:   workerNicIds,
			ControlNicIds:  controlNicIds,
			StorageAccUri:  storageAcc.PrimaryEndpoints.Blob(),
			SubnetID:       networkResources.Vnet.Subnets.Index(pulumi.Int(0)).Id(),
			NsgId:          networkResources.NetworkSecurityGroup.ID(),
		})
		if err != nil {
			return err
		}

		nicOutputs := make([]interface{}, len(networkResources.ControlNetworkInterfaces))
		for i, nic := range networkResources.ControlNetworkInterfaces {
			nicIp := networkResources.NetworkInterfacePublicIPs[i].IpAddress
			nicOutput := pulumi.All(nic.Name, nicIp).ApplyT(
				func(args []interface{}) map[string]interface{} {
					name := args[0].(string)
					ipAddress := args[1].(*string)
					return map[string]interface{}{
						"name": name,
						"ip":   *ipAddress,
					}
				},
			)
			nicOutputs[i] = nicOutput
		}
		nicOut := pulumi.All(nicOutputs...).ApplyT(
			func(args []interface{}) []interface{} {
				return args
			}).(pulumi.ArrayOutput)

		clusterClientCfg.TalosConfig().ApplyT(func(cfg string) (string, error) {
			d1 := []byte(cfg)
			err := os.WriteFile("secrets/talosconfig", d1, 0644)
			if err != nil {
				return "", err
			}

			return "ok", nil
		})

		ctx.Export("NetworkInterfaces", nicOut)
		ctx.Export("Vnet.Name", networkResources.Vnet.Name)
		ctx.Export("PublicIp.IpAddress", networkResources.PublicLbIp.IpAddress)
		ctx.Export("PublicNatIp.IpAddress", networkResources.PublicNatIp.IpAddress)
		ctx.Export("LoadBalancer.IpAddress", networkResources.PublicLbIp.IpAddress)
		ctx.Export("clusterClientCfg", clusterClientCfg.TalosConfig())
		ctx.Export("storageAccount.Name", storageAcc.Name)

		return nil
	})
}
