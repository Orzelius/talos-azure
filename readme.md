# Talon linux on azure via pulumi (golang)

A simple project that follows the official Talos [Creating a cluster via the CLI on Azure](https://www.talos.dev/v1.7/talos-guides/install/cloud-platforms/azure/#network-infrastructure) guide to set
up a talos cluster on azure with Pulumi via the [azure-native](https://www.pulumi.com/registry/packages/azure-native/) and [Talos](https://www.pulumi.com/registry/packages/talos/) providers.

## Instructions

### Prerequisites

* [talosctl installed](https://www.talos.dev/v1.7/talos-guides/install/talosctl/)
* [pulumi installed](https://www.pulumi.com/docs/clouds/azure/get-started/begin/#install-pulumi)
* Azure account with sufficient permissions

### Instructions

1. Add your config

```sh
cp example.Pulumi.dev.yaml Pulumi.dev.yaml
```

Make sure to change the region and machine number values.

2. Authentivate to azure and configure account.

See pulumi documentation: [Azure Native: Installation & Configuration](https://www.pulumi.com/registry/packages/azure-native/installation-configuration/#azure-native-installation-configuration)

3. Deploy the services

```sh
pulumi up
```

4. Setup the cluster

```sh
sh setup-cluster.sh
```

optionally check cluster health via 

```sh
talosctl --talosconfig secrets/talosconfig health
```

5. Get the kubcetl config

```sh
talosctl --talosconfig secrets/talosconfig kubeconfig secrets/kubeconfig
```

test out kube config

```sh
kubectl --kubeconfig secrets/kubeconfig get nodes
```

6. Clean up

Once you're done with the cluster you can delete the resources with

```sh
pulumi destroy
```
