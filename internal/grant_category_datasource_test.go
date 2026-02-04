package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGrantCategoryDataSource(t *testing.T) {
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
				Config: providerConfig + `
data "collibra-data-access_grant_category" "test" {
	name = "Role"
}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "id", "default"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "name", "Role"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "name_plural", "Roles"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "is_system", "false"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "is_default", "true"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "can_create", "true"),
					resource.TestCheckResourceAttr("data.collibra-data-access_grant_category.test", "allow_duplicate_names", "true"),
				),
			},
		},
	})

}
