package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
	raitoType "github.com/raito-io/sdk-go/types"
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
resource "raito_datasource" "test" {
	name        = "tfTestDataSource-%s"
	description = "test description"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodOnPrem)),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "0"),
						resource.TestCheckNoResourceAttr("raito_datasource.test", "parent"),
						resource.TestCheckResourceAttrWith("raito_datasource.test", "native_identity_store", func(value string) error {
							if value == "" {
								return errors.New("native_identity_store should not be empty")
							}

							return nil
						}),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_datasource" "test" {
	name        = "tfTestDataSourceUpdateName-%s"
	description = "test update description"
	sync_method = "CLOUD_MANUAL_TRIGGER"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSourceUpdateName-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test update description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodCloudManualTrigger)),
					),
				},
				// Resource are automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})

	t.Run("link_identity_stores", func(t *testing.T) {
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
resource "raito_identitystore" "tfIs1" {
	name = "tfIs1DataSourceTest-%[1]s"
}

resource "raito_identitystore" "tfIs2" {
    name = "tfIs2DataSourceTest-%[1]s"
}

resource "raito_datasource" "test" {
	name = "tfDs1-%[1]s"
	identity_stores = [
		raito_identitystore.tfIs1.id,
	]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfDs1-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.tfIs1", "id", "raito_datasource.test", "identity_stores.0"),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_identitystore" "tfIs1" {
	name = "tfIs1DataSourceTest-%[1]s"
}

resource "raito_identitystore" "tfIs2" {
    name = "tfIs2DataSourceTest-%[1]s"
}

resource "raito_datasource" "test" {
	name = "tfDs1-%[1]s"
	identity_stores = [
		raito_identitystore.tfIs2.id,
	]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfDs1-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.tfIs2", "id", "raito_datasource.test", "identity_stores.0"),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
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
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "test_user" {
  name       = "TestUser%[1]s"
  email      = "test_user-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_datasource" "test" {
	name        = "tfTestDataSource-%[1]s"
	description = "test description"
	owners      = [ raito_user.test_user.id ]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodOnPrem)),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "0"),
						resource.TestCheckNoResourceAttr("raito_datasource.test", "parent"),
						resource.TestCheckResourceAttrWith("raito_datasource.test", "native_identity_store", func(value string) error {
							if value == "" {
								return errors.New("native_identity_store should not be empty")
							}

							return nil
						}),
						resource.TestCheckResourceAttr("raito_datasource.test", "owners.#", "1"),
						resource.TestCheckResourceAttrPair("raito_datasource.test", "owners.0", "raito_user.test_user", "id"),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "test_user" {
  name       = "TestUser%[1]s"
  email      = "test_user-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_user" "test_user_2" {
  name       = "TestUser-2-%[1]s"
  email      = "test_user-2-%[1]s@raito.io"
  raito_user = true
  type       = "Machine"					
}
					
resource "raito_datasource" "test" {
	name        = "tfTestDataSource-%[1]s"
	description = "test description"
	owners      = [ raito_user.test_user_2.id ]
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSource-"+testId),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodOnPrem)),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "0"),
						resource.TestCheckNoResourceAttr("raito_datasource.test", "parent"),
						resource.TestCheckResourceAttrWith("raito_datasource.test", "native_identity_store", func(value string) error {
							if value == "" {
								return errors.New("native_identity_store should not be empty")
							}

							return nil
						}),
						resource.TestCheckResourceAttr("raito_datasource.test", "owners.#", "1"),
						resource.TestCheckResourceAttrPair("raito_datasource.test", "owners.0", "raito_user.test_user_2", "id"),
					),
				},
				// Resource are automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})
}
