package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGrantResource(t *testing.T) {
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
					Config: providerConfig + `
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
		}
	]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "category", "default"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_grant.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	state = "Inactive"
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
			permissions: ["SELECT"]
		}
	]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
	inheritance_locked = true
}
`),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.permissions.0", "SELECT"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	state = "Inactive"
	what_locked = true
	who_locked = true
	inheritance_locked = true
}
`),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.test", "what_data_objects"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.test", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	state = "Inactive"
	what_locked = false
	who_locked = false
	inheritance_locked = false
}
`),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.test", "what_data_objects"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.test", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "false"),
					),
				},
			},
		})
	})

	t.Run("grant with purposes", func(t *testing.T) {
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
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
	category = "purpose"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
		}
	]
	who = [
		{
			"access_control": collibra-access-governance_grant.purpose1.id
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "who.0.access_control", "collibra-access-governance_grant.purpose1", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + `
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_grant" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
	category = "purpose"
}

resource "collibra-access-governance_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
		}
	]
	who = [
		{
			"access_control": collibra-access-governance_grant.purpose1.id
		},
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.test", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who.#", "2"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.test", "what_locked", "true"),
					),
				},
			},
		})
	})

	t.Run("what abac", func(t *testing.T) {
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
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		literal = true
	})
}

resource "collibra-access-governance_grant" "abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_abac_rule = {
        rule = local.abac_rule
		do_types = ["table"]
		scope = [
			{
				data_source: data.collibra-access-governance_datasource.ds.id
				fullname: "MASTER_DATA"
			}
		]
    }
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.abac_grant", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.abac_grant", "what_data_objects"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.scope.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.scope.0.fullname", "MASTER_DATA"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.abac_grant", "what_abac_rule.scope.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.global_permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.global_permissions.0", "READ"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_grant.abac_grant",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + `
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		literal = true
	})
}

resource "collibra-access-governance_grant" "abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_abac_rule = {
        rule = local.abac_rule
		scope = [
			{
				fullname: "MASTER_DATA.PERSON"
				data_source: data.collibra-access-governance_datasource.ds.id
			},
			{
				fullname: "MASTER_DATA.SALES"
				data_source: data.collibra-access-governance_datasource.ds.id
			}
		]
		global_permissions = ["WRITE"]
		permissions = ["SELECT"]
		do_types = ["table", "view"]
    }
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.abac_grant", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.abac_grant", "what_data_objects"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.scope.#", "2"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.global_permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.global_permissions.0", "WRITE"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.permissions.0", "SELECT"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_abac_rule.do_types.#", "2"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.abac_grant", "what_locked", "true"),
					),
				},
			},
		})
	})

	t.Run("who abac rule", func(t *testing.T) {
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
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		aggregator: {
			operator: "Or",
			operands: [
				{
					aggregator: {
						operator: "And",
						operands: [
							{
								comparison: {
									operator: "HasTag"
									leftOperand: "Test"
									rightOperand: {
										literal: { string: "test" }
									}
								}
							}
						]
					}
				}
			]
		}
	})
}

resource "collibra-access-governance_grant" "who_abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
		}
	]
	who_abac_rule = local.abac_rule
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.who_abac_grant", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.who_abac_grant", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "who_abac_rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_grant.who_abac_grant",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + `
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		aggregator: {
			operator: "Or",
			operands: [
				{
					aggregator: {
						operator: "And",
						operands: [
							{
								comparison: {
									operator: "HasTag"
									leftOperand: "Test"
									rightOperand: {
										literal: { string: "test" }
									}
								}
							}
						]
					}
				}
			]
		}
	})
}

resource "collibra-access-governance_grant" "who_abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = [
		{  
			data_source = data.collibra-access-governance_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			fullname = "MASTER_DATA.SALES"
			data_source = data.collibra-access-governance_datasource.ds.id
		}
	]
	who_abac_rule = local.abac_rule
	inheritance_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_grant.who_abac_grant", "data_source.0.data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_grant.who_abac_grant", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "who_abac_rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_grant.who_abac_grant", "what_locked", "true"),
					),
				},
			},
		})
	})
}
