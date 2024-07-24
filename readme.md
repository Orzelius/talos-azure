# Talon Linux on Azure via Pulumi (golang)

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

## Takeaways

### Azure and Pulumi

I very much like pulumi and the ide of IaC in general. I think it's the right way forward
and a much better solution for serious projects compared to doing things with the CLI or GUI.

That being said the experience with Azure while using pulumi Pulumi is not great. The API behavior is often different from the cli and the GUI. API error messages are a bit hard to read and there
is no documentation for behavioral differences. A lot of trial and error is required. I found working with gCloud a much more pleasant experience. The terraform code provided by google is close to 1 to 1 to the equivalent Pulumi code and the behavior is much more in sync between the API, UI and API.

### Pulumi and golang

I have nothing agains golang and have used it in both professional and personal settings, but for Pulumi I'd opt for another language. I've previously worked on a very similar project to this one, but instead used Typescript and found the experience much more pleasant.

Just comparing the official pulumi code snippets will give you the idea, but golang code is times more verbose and boilerplate-y than the TS equivalent. Not to mention the lack of handy helpers and flexible types offered by TS. I think the simplicity and verbosity golang offers (or rather enforces) is just hurting things when working with very dynamic and flexible code like you do in case of Pulumi.
