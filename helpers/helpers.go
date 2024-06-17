package helpers

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type CustomConfig struct {
	AzRegion string
}

func GetConfig(ctx *pulumi.Context) (CustomConfig, error) {
	azConf := config.New(ctx, "azure-native")
	if azConf == nil {
		return CustomConfig{}, getConfNotFoundErr("azure", "top level")
	}

	azRegion := azConf.Require("location")
	if azRegion == "" {
		return CustomConfig{}, getConfNotFoundErr("azure", "location")
	}

	return CustomConfig{AzRegion: azRegion}, nil
}

func getConfNotFoundErr(prefix string, confName string) error {
	return fmt.Errorf("%s %s configuration not set", prefix, confName)
}
