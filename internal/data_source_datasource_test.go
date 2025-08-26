package internal

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/raito-io/sdk-go/types"
)

func TestAccDataSourceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			AccProviderPreCheck(t)
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "raito_datasource" "test" {
    name = "Snowflake"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.raito_datasource.test", "name", "Snowflake"),
					resource.TestCheckResourceAttrWith("data.raito_datasource.test", "id", func(value string) error {
						if value == "" {
							return errors.New("ID is not set")
						}

						return nil
					}),
					resource.TestCheckResourceAttr("data.raito_datasource.test", "sync_method", string(types.DataSourceSyncMethodOnPrem)),
					resource.TestCheckResourceAttrSet("data.raito_datasource.test", "owners.0"),
				),
			},
		},
	})
}
