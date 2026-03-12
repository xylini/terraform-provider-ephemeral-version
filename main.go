package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/xylini/terraform-provider-ephemeralversion/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/xylini/ephemeralversion",
	})
	if err != nil {
		log.Fatal(err)
	}
}
