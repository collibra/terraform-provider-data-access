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
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_mask" "test" {
	name        = "tfTestMask"
	type        = "NULL"
    description = "test description"
	data_source = data.collibra-access-governance_datasource.ds.id
	columns = []
	who = [
     {
       user             = "terraform@collibra.com"
       promise_duration = 604800
     }
   ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_mask.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "columns.#", "0"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who.0.user", "terraform@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "type", "NULL"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_mask.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "columns"},
				},
				{
					Config: providerConfig + `data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_mask" "test" {
	name        = "Terraform Mask name edit"
	type        = "NULL"
    description = "test description"
	data_source = data.collibra-access-governance_datasource.ds.id
	who = [
     {
       user             = "terraform@collibra.com"
     }
   ]
	inheritance_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_mask.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "columns"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who.0.user", "terraform@collibra.com"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "who.0.promise_duration"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "type", "NULL"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "inheritance_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "what_locked", "false"),
					),
				},
				{
					Config: providerConfig + `data "collibra-access-governance_datasource" "ds" {
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

resource "collibra-access-governance_mask" "test" {
	name        = "Terraform Mask name edit"
	type        = "NULL"
    description = "test description"
	data_source = data.collibra-access-governance_datasource.ds.id
	who_abac_rule = local.abac_rule
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_mask.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "columns"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_abac_rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "type", "NULL"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "what_locked", "false"),
					),
				},
				{
					Config: providerConfig + `data "collibra-access-governance_datasource" "ds" {
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

resource "collibra-access-governance_mask" "test" {
	name        = "Terraform Mask name edit"
	type        = "NULL"
    description = "test description"
	data_source = data.collibra-access-governance_datasource.ds.id
	who_abac_rule = local.abac_rule
	what_locked = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_mask.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "columns"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.test", "who"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_abac_rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "type", "NULL"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.test", "what_locked", "true"),
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

resource "collibra-access-governance_mask" "abac_mask" {
	name        = "tfTestMask"
	type        = "NULL"
    description = "test description"
	data_source = data.collibra-access-governance_datasource.ds.id
	who = [
	     {
	       user             = "terraform@collibra.com"
	       promise_duration = 604800
	     }
    ]
	what_abac_rule = {
		rule = local.abac_rule
		scope = ["MASTER_DATA.PERSON", "MASTER_DATA.SALES"]
	}
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "description", "test description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_mask.abac_mask", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_mask.abac_mask", "columns"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "what_abac_rule.scope.#", "2"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "who.0.user", "terraform@collibra.com"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "type", "NULL"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "what_abac_rule.rule", "{\"literal\":true}"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_mask.abac_mask", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_mask.abac_mask",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "columns"},
				},
			},
		})
	})
}
