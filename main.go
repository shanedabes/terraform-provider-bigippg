package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/shanedabes/terraform-provider-bigippg/bigippg"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: bigippg.Provider})
}
