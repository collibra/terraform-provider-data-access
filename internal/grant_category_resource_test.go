package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccGrantCategoryResource(t *testing.T) {
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
resource "raito_grant_category" "test" {
	name        = "tfTestGrantCategory-%s"
	description = "test description"
	icon		= "test"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_grant_category.test", "name", "tfTestGrantCategory-"+testId),
						resource.TestCheckResourceAttr("raito_grant_category.test", "description", "test description"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "is_system", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "is_default", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "can_create", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allow_duplicate_names", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "multi_data_source", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "default_type_per_data_source.#", "0"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.user", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.group", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.inheritance", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.self", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.categories.#", "0"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_what_items.data_object", "true"),
					),
				},
				{
					ResourceName:      "raito_grant_category.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_grant_category" "test" {
	name        = "tfTestGrantCategory-%s"
	description = "test description update"
	icon		= "test"
	allow_duplicate_names = false
	multi_data_source = false
	allowed_who_items = {
		user = false
	}
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_grant_category.test", "name", "tfTestGrantCategory-"+testId),
						resource.TestCheckResourceAttr("raito_grant_category.test", "description", "test description update"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "is_system", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "is_default", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "can_create", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allow_duplicate_names", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "multi_data_source", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "default_type_per_data_source.#", "0"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.user", "false"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.group", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.inheritance", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.self", "true"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_who_items.categories.#", "0"),
						resource.TestCheckResourceAttr("raito_grant_category.test", "allowed_what_items.data_object", "true"),
					),
				},
				// Resource is automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})
}
