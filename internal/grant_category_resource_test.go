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
resource "collibra-data-access_grant_category" "test" {
	name        = "tfTestGrantCategory-%[1]s"
    name_plural = "tfTestGrantCategories-%[1]s"
	description = "test description"
	icon		= "test"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "name", "tfTestGrantCategory-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "description", "test description"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "is_system", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "is_default", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "can_create", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allow_duplicate_names", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "multi_data_source", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "default_type_per_data_source.#", "0"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.user", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.inheritance", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.self", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.categories.#", "0"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_what_items.data_object", "true"),
					),
				},
				{
					ResourceName:      "collibra-data-access_grant_category.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "collibra-data-access_grant_category" "test" {
	name        = "tfTestGrantCategory-%[1]s"
    name_plural = "tfTestGrantCategories-%[1]s"
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
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "name", "tfTestGrantCategory-"+testId),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "description", "test description update"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "is_system", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "is_default", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "can_create", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allow_duplicate_names", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "multi_data_source", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "default_type_per_data_source.#", "0"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.user", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.inheritance", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.self", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_who_items.categories.#", "0"),
						resource.TestCheckResourceAttr("collibra-data-access_grant_category.test", "allowed_what_items.data_object", "true"),
					),
				},
				// Resource is automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})
}
