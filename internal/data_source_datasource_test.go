package internal

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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
				Config: providerConfig + `data "collibra-data-access_datasource" "test" {
    name = "Snowflake"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.collibra-data-access_datasource.test", "name", "Snowflake"),
					resource.TestCheckResourceAttrWith("data.collibra-data-access_datasource.test", "id", func(value string) error {
						if value == "" {
							return errors.New("ID is not set")
						}

						return nil
					}),
					resource.TestCheckResourceAttrSet("data.collibra-data-access_datasource.test", "owners.0"),
				),
			},
		},
	})
}
