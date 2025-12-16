package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccDataSourceResource(t *testing.T) {
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
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "collibra-data-access_datasource" "test" {
	name        = "tfTestDataSource-%s"
	description = "test description"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "description", "test description"),
						resource.TestCheckNoResourceAttr("collibra-data-access_datasource.test", "parent"),
					),
				},
				{
					ResourceName:      "collibra-data-access_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "collibra-data-access_datasource" "test" {
	name        = "tfTestDataSourceUpdateName-%s"
	description = "test update description"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "name", "tfTestDataSourceUpdateName-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "description", "test update description"),
					),
				},
				// Resource are automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})

	t.Run("set owners", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_0_0),
			},
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_user" "acc-user-1" {
  email = "%[2]s"
}

resource "collibra-data-access_datasource" "test" {
	name        = "tfTestDataSource-%[1]s"
	description = "test description"
	owners      = [ data.collibra-data-access_user.acc-user-1.id ]
}
`, testId, TestUser1Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "description", "test description"),
						resource.TestCheckNoResourceAttr("collibra-data-access_datasource.test", "parent"),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "owners.#", "1"),
						//resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "owners.0", TestUser1Id),
					),
				},
				{
					ResourceName:      "collibra-data-access_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_user" "acc-user-2" {
  email = "%[2]s"
}

resource "collibra-data-access_datasource" "test" {
	name        = "tfTestDataSource-%[1]s"
	description = "test description"
	owners      = [ data.collibra-data-access_user.acc-user-2.id ]
}
`, testId, TestUser2Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "description", "test description"),
						resource.TestCheckNoResourceAttr("collibra-data-access_datasource.test", "parent"),
						resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "owners.#", "1"),
						//resource.TestCheckResourceAttr("collibra-data-access_datasource.test", "owners.0", TestUser2Id),
					),
				},
				// Resource are automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})
}
