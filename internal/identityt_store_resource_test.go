package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccIdentityStoreResource(t *testing.T) {
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
					Config: providerConfig + fmt.Sprintf(`
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore-%s"
	description = "terraform test identity store"
    master = false
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore-"+testId),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "false"),
					),
				},
				{
					ResourceName:      "raito_identitystore.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore-Rename-%s"
	description = "terraform test identity store renaming description"
    master = true
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore-Rename-"+testId),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store renaming description"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "true"),
					),
				},
			},
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
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "test_user" {
  name       = "IsTestUser%[1]s"
  email      = "is_test_user-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore-%[1]s"
	description = "terraform test identity store"
    master = false
	owners = [ raito_user.test_user.id ]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore-"+testId),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "false"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "owners.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.test", "owners.0", "raito_user.test_user", "id"),
					),
				},
				{
					ResourceName:      "raito_identitystore.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "test_user" {
  name       = "IsTestUser%[1]s"
  email      = "is_test_user-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_user" "test_user_2" {
  name       = "IsTestUser2%[1]s"
  email      = "is_test_user-2-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore-%[1]s"
	description = "terraform test identity store"
    master = false
	owners = [ raito_user.test_user_2.id ]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore-"+testId),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "false"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "owners.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.test", "owners.0", "raito_user.test_user_2", "id"),
					),
				},
			},
		})
	})
}
