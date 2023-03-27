package main

import (
	"fmt"

	"github.com/Instawork/terraform-provider-imgixyz/internal"
)

func main() {
	// plugin.Serve(&plugin.ServeOpts{
	// 	ProviderFunc: func() *schema.Provider {
	// 		return internal.Provider()
	// 	},
	// })
	client := internal.ImgixClient{}
	client.SetAuthToken("***REMOVED***")
	resp, err := client.GetSourceByID("63059169e12941a3897eca9e")
	fmt.Println(resp)
	fmt.Println(err)
}
