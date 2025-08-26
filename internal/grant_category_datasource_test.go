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
data "raito_grant_category" "test" {
	name = "Purpose"
}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "id", "purpose"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "name", "Purpose"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "is_system", "false"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "is_default", "false"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "can_create", "true"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "allow_duplicate_names", "true"),
					resource.TestCheckResourceAttr("data.raito_grant_category.test", "default_type_per_data_source.#", "0"),
				),
			},
		},
	})

}
