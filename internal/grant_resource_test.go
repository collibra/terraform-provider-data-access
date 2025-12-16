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
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
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
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.data_object.path.1", "SALES"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "category", "default"),
					),
				},
				{
					ResourceName:            "collibra-data-access_grant.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	state = "Inactive"
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
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
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.data_object.path.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.data_object.path.0", "MASTER_DATA"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.permissions.0", "SELECT"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
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
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.test", "what_data_objects"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.test", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
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
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.test", "what_data_objects"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.test", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "false"),
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
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
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

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
		}
	]
	who = [
		{
			"access_control": collibra-data-access_grant.purpose1.id
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.data_object.path.0", "MASTER_DATA"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "who.0.access_control", "collibra-data-access_grant.purpose1", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "true"),
					),
				},
				{
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_grant" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
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

resource "collibra-data-access_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
		}
	]
	who = [
		{
			"access_control": collibra-data-access_grant.purpose1.id
		},
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_data_objects.0.data_object.path.1", "SALES"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.test", "what_locked", "true"),
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
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		literal = true
	})
}

resource "collibra-data-access_grant" "abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_abac_rules = [{
		id = "rule1"
		rule = local.abac_rule
		do_types = ["table"]
		scope = [
			{
				data_source: data.collibra-data-access_datasource.ds.id
				type: "database"
				path: ["MASTER_DATA"]
			}
		]
    }]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.abac_grant", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.abac_grant", "what_data_objects"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.scope.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.scope.0.path.0", "MASTER_DATA"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.abac_grant", "what_abac_rules.0.scope.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.global_permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.global_permissions.0", "READ"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_grant.abac_grant",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		literal = true
	})
}

resource "collibra-data-access_grant" "abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_abac_rules = [{
		id = "rule1"	
		rule = local.abac_rule
		scope = [
			{
				type: "schema"
				path: ["MASTER_DATA", "PERSON"]
				data_source: data.collibra-data-access_datasource.ds.id
			},
			{
				type: "schema"
				path: ["MASTER_DATA", "SALES"]
				data_source: data.collibra-data-access_datasource.ds.id
			}
		]
		global_permissions = ["WRITE"]
		permissions = ["SELECT"]
		do_types = ["table", "view"]
    }]
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.abac_grant", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.abac_grant", "what_data_objects"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.scope.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.global_permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.global_permissions.0", "WRITE"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.permissions.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.permissions.0", "SELECT"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_abac_rules.0.do_types.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.abac_grant", "what_locked", "true"),
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
data "collibra-data-access_datasource" "ds" {
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

resource "collibra-data-access_grant" "who_abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
		}
	]
	who_abac_rules = [
		{
			id = "rule1"
			rule = local.abac_rule
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.who_abac_grant", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_data_objects.0.data_object.type", "schema"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.who_abac_grant", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "who_abac_rules.0.rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_grant.who_abac_grant",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
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

resource "collibra-data-access_grant" "who_abac_grant" {
	name        = "tfTestGrant"
    description = "test description"
	data_sources = [
		{  
			data_source = data.collibra-data-access_datasource.ds.id
			type = "role"
		}
	]
	what_data_objects = [
		{
			data_object = {
				type = "schema"
				path = ["MASTER_DATA", "SALES"]
				data_source = data.collibra-data-access_datasource.ds.id
			}
		}
	]
	who_abac_rules = [
		{
			id = "rule1"
			rule = local.abac_rule
		}
	]
	inheritance_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_grant.who_abac_grant", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_data_objects.0.data_object.path.0", "MASTER_DATA"),
						resource.TestCheckNoResourceAttr("collibra-data-access_grant.who_abac_grant", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "who_abac_rules.0.rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_grant.who_abac_grant", "what_locked", "true"),
					),
				},
			},
		})
	})
}
