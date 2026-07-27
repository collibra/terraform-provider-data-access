package internal

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// These users exist on the "Global Okta" data source, so they can be used in the who
// component of a group targeting that data source.
const (
	groupTestOktaUser1Email = "test.user.7zvwyszl@collibra.dev"
	groupTestOktaUser2Email = "test.user.ew6v3vju@collibra.dev"
)

func TestAccGroupResource(t *testing.T) {
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
data "collibra-data-access_datasource" "ds" {
    name = "Global Okta"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroup"
    description = "test description"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
	who = [
		{
			"user": "%s"
		}
	]
}
`, groupTestOktaUser1Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "name", "tfTestGroup"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_group.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who.0.user", groupTestOktaUser1Email),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "inheritance_locked", "false"),
					),
				},
				{
					ResourceName:            "collibra-data-access_group.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who"},
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_datasource" "ds" {
    name = "Global Okta"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroup"
    description = "test description updated"
	state       = "Inactive"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
	who = [
		{
			"user": "%s"
		}
	]
	inheritance_locked = true
}
`, groupTestOktaUser2Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "name", "tfTestGroup"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "description", "test description updated"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "state", "Inactive"),
						resource.TestCheckResourceAttrPair("collibra-data-access_group.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who.0.user", groupTestOktaUser2Email),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "inheritance_locked", "true"),
					),
				},
				{
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Global Okta"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroup"
    description = "test description updated"
	state       = "Inactive"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
	who_locked         = false
	inheritance_locked = false
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckNoResourceAttr("collibra-data-access_group.test", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "inheritance_locked", "false"),
					),
				},
			},
		})
	})

	t.Run("owners", func(t *testing.T) {
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
data "collibra-data-access_datasource" "ds" {
    name = "Global Okta"
}

data "collibra-data-access_user" "acc-user-1" {
  email = "%s"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroupOwners"
	description = "test description"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
	owners = [ data.collibra-data-access_user.acc-user-1.id ]
}
`, TestUser1Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "name", "tfTestGroupOwners"),
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "owners.#", "1"),
					),
				},
				{
					ResourceName:      "collibra-data-access_group.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_datasource" "ds" {
    name = "Global Okta"
}

data "collibra-data-access_user" "acc-user-2" {
  email = "%s"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroupOwners"
	description = "test description"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
	owners = [ data.collibra-data-access_user.acc-user-2.id ]
}
`, TestUser2Email),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_group.test", "owners.#", "1"),
					),
				},
			},
		})
	})

	t.Run("unsupported data source", func(t *testing.T) {
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
					// Snowflake has no concept of groups, so Collibra Data Access rejects
					// the group action for it.
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_group" "test" {
	name        = "tfTestGroupUnsupported"
	description = "test description"
	data_sources = [
		{
			data_source = data.collibra-data-access_datasource.ds.id
		}
	]
}
`,
					ExpectError: regexp.MustCompile(`Failed to create access provider`),
				},
			},
		})
	})
}
