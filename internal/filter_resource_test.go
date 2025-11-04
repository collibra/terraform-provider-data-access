package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccFilterResource(t *testing.T) {
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
					ResourceName: "collibra-data-access_filter.test",
					Config: providerConfig + `
				data "collibra-data-access_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "collibra-data-access_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.collibra-data-access_datasource.ds.id
					table = "MASTER_DATA.SALES.SPECIALOFFER"
					who = [
						{
							"user": "terraform-acc-test-1@collibra.com"
						}
					]
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "table", "MASTER_DATA.SALES.SPECIALOFFER"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_filter.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "table"},
				},
				{
					ResourceName: "collibra-data-access_filter.test",
					Config: providerConfig + `
				data "collibra-data-access_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "collibra-data-access_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.collibra-data-access_datasource.ds.id
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_filter.test", "table"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "what_locked", "false"),
					),
				},
				{
					ResourceName: "collibra-data-access_filter.test",
					Config: providerConfig + `
data "collibra-data-access_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-data-access_filter" "test" {
	name        = "tfTestFilter"
    description = "filter description"
	data_source = data.collibra-data-access_datasource.ds.id
	filter_policy = "{Category} = 'Reseller'"
	what_locked = false
	who = [
		{
			"user": "terraform-acc-test-1@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-data-access_filter.test", "table"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "what_locked", "false"),
					),
				},
			},
		})
	})
}
