package azuredevops

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var provider = Provider()

// TestAzureDevOpsProvider_foo
func init() {
	InitProvider()
}

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func InitProvider() {
	testAccProvider = provider
	testAccProviders = map[string]terraform.ResourceProvider{
		"azuredevops": testAccProvider,
	}
}
