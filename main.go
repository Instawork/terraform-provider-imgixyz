package main

import (
	"context"
	"log"

	"github.com/Instawork/terraform-provider-imgixyz/internal"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/Instawork/imgixyz",
	}
	err := providerserver.Serve(context.Background(), internal.NewProvider("0.1"), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
