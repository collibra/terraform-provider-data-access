package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccDataSourceDataSource(t *testing.T) {
	testId := gonanoid.Must(8)

	t.Run("basic", func(t *testing.T) {
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
	})

	t.Run("type is returned when set on create", func(t *testing.T) {
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
					Config: providerConfig + fmt.Sprintf(`
resource "collibra-data-access_datasource" "test" {
	name = "tfTestDataSource-%s"
	type = "Snowflake"
}

data "collibra-data-access_datasource" "test" {
	name       = collibra-data-access_datasource.test.name
	depends_on = [collibra-data-access_datasource.test]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.collibra-data-access_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("data.collibra-data-access_datasource.test", "type", "Snowflake"),
					),
				},
			},
		})
	})
}
