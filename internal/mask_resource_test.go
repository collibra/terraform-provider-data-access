package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccMaskResource(t *testing.T) {
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

resource "collibra-data-access_mask" "test" {
	name        = "tfTestMask"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	columns = []
	who = [
     {
       user             = "terraform-acc-test-1@collibra.com"
       promise_duration = 604800
     }
   ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "data_sources.0.type", "NULL"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "columns.#", "0"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_mask.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "columns"},
				},
				{
					Config: providerConfig + `data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_mask" "test" {
	name        = "Terraform Mask name edit"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	who = [
     {
       user             = "terraform-acc-test-1@collibra.com"
     }
   ]
	inheritance_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "columns"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "who.0.promise_duration"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "what_locked", "false"),
					),
				},
				{
					Config: providerConfig + `data "collibra-data-access_datasource" "ds" {
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

resource "collibra-data-access_mask" "test" {
	name        = "Terraform Mask name edit"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	who_abac_rules = [
		{
			id = "rule1"
			rule = local.abac_rule
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "data_sources.0.type", "NULL"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "columns"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_abac_rules.0.rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "what_locked", "false"),
					),
				},
				{
					Config: providerConfig + `data "collibra-data-access_datasource" "ds" {
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

resource "collibra-data-access_mask" "test" {
	name        = "Terraform Mask name edit"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	who_abac_rules = [
		{
			id = "rule1"
			rule = local.abac_rule
		}
	]
	what_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.test", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "data_sources.0.type", "NULL"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "columns"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.test", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_abac_rules.0.rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.test", "what_locked", "true"),
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

resource "collibra-data-access_mask" "abac_mask" {
	name        = "tfTestMask"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	who = [
	     {
	       user             = "terraform-acc-test-1@collibra.com"
	       promise_duration = 604800
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
	}]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.abac_mask", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "data_sources.0.type", "NULL"),
						resource.TestCheckNoResourceAttr("collibra-data-access_mask.abac_mask", "columns"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.0.scope.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.0.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_mask.abac_mask",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "columns"},
				},
				{
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule1 = jsonencode({
		literal = true
	})

	abac_rule2 = jsonencode({
		literal = false
	})
}

resource "collibra-data-access_mask" "abac_mask" {
	name        = "tfTestMask"
	description = "test description"
	data_sources = [{
		data_source = data.collibra-data-access_datasource.ds.id
		type = "NULL"
	}]
	who = [
	     {
	       user             = "terraform-acc-test-1@collibra.com"
	       promise_duration = 604800
	     }
    ]
	what_abac_rules = [{
		id = "rule1"
		rule = local.abac_rule1
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
	},
	{
		id = "rule2"
		rule = local.abac_rule2
		scope = [
			{
				type: "schema"
				path: ["MASTER_DATA", "SALES"]
				data_source: data.collibra-data-access_datasource.ds.id
			}
		]
	}]
	columns = [
		{
			type: "column"
			path: ["MASTER_DATA", "SALES", "CREDITCARD", "CardNumber"]
			data_source: data.collibra-data-access_datasource.ds.id
		},
		{
			type: "column"
			path: ["MASTER_DATA", "PERSON", "ADDRESS", "AddressLine1"]
			data_source: data.collibra-data-access_datasource.ds.id
		}]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_mask.abac_mask", "data_sources.0.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "data_sources.0.type", "NULL"),

						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.0.id", "rule1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.0.scope.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.1.id", "rule2"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.1.scope.#", "1"),

						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "columns.#", "2"),

						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_abac_rules.0.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_mask.abac_mask", "what_locked", "true"),
					),
				},
			},
		})
	})
}
