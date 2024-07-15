package helpers

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type CustomConfig struct {
	AzRegion     string
	WorkerCount  int
	ControlCount int
	Architecture string
	TalosVersion string
	ClusterName  string
	Vm           string
}

func GetConfig(ctx *pulumi.Context) (CustomConfig, error) {
	clusterCfg := config.New(ctx, "cluster")
	if clusterCfg == nil {
		return CustomConfig{}, getConfNotFoundErr("cluster", "top level")
	}
	azConf := config.New(ctx, "azure-native")
	if azConf == nil {
		return CustomConfig{}, getConfNotFoundErr("azure", "top level")
	}

	azRegion := azConf.Require("location")
	if azRegion == "" {
		return CustomConfig{}, getConfNotFoundErr("azure", "location")
	}

	workerCountS := clusterCfg.Require("workers")
	if workerCountS == "" {
		return CustomConfig{}, getConfNotFoundErr("azure", "workers")
	}
	workerCount, err := strconv.Atoi(workerCountS)
	if err != nil {
		return CustomConfig{}, fmt.Errorf("cluster:workers config must be an integer, %e", err)
	}
	controlCountS := clusterCfg.Require("controls")
	if controlCountS == "" {
		return CustomConfig{}, getConfNotFoundErr("cluster", "controls")
	}
	controlCount, err := strconv.Atoi(controlCountS)
	if err != nil {
		return CustomConfig{}, fmt.Errorf("cluster:workers config must be an integer, %e", err)
	}

	arc := clusterCfg.Require("architecture")
	if arc == "" {
		return CustomConfig{}, getConfNotFoundErr("cluster", "architecture")
	}

	talosVer := clusterCfg.Require("talos-version")
	if talosVer == "" {
		return CustomConfig{}, getConfNotFoundErr("cluster", "talos-version")
	}

	name := clusterCfg.Require("name")
	if name == "" {
		return CustomConfig{}, getConfNotFoundErr("cluster", "name")
	}

	vm := clusterCfg.Require("vm")
	if name == "" {
		return CustomConfig{}, getConfNotFoundErr("cluster", "vm")
	}

	return CustomConfig{
		AzRegion:     azRegion,
		WorkerCount:  workerCount,
		ControlCount: controlCount,
		Architecture: arc,
		TalosVersion: talosVer,
		ClusterName:  name,
		Vm:           vm,
	}, nil
}

func getConfNotFoundErr(prefix string, confName string) error {
	return fmt.Errorf("%s %s configuration not set", prefix, confName)
}
