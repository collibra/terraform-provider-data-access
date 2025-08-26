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
					ResourceName: "raito_filter.test",
					Config: providerConfig + `
				data "raito_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "raito_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.raito_datasource.ds.id
					table = "MASTER_DATA.SALES.SPECIALOFFER"
					who = [
						{
							"user": "terraform@raito.io"
						}
					]
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("raito_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("raito_filter.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckResourceAttr("raito_filter.test", "table", "MASTER_DATA.SALES.SPECIALOFFER"),
						resource.TestCheckResourceAttr("raito_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("raito_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("raito_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("raito_filter.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "raito_filter.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "table"},
				},
				{
					ResourceName: "raito_filter.test",
					Config: providerConfig + `
				data "raito_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "raito_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.raito_datasource.ds.id
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("raito_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("raito_filter.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("raito_filter.test", "table"),
						resource.TestCheckResourceAttr("raito_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("raito_filter.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("raito_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("raito_filter.test", "what_locked", "false"),
					),
				},
				{
					ResourceName: "raito_filter.test",
					Config: providerConfig + `
data "raito_datasource" "ds" {
    name = "Snowflake"
}

resource "raito_filter" "test" {
	name        = "tfTestFilter"
    description = "filter description"
	data_source = data.raito_datasource.ds.id
	filter_policy = "{Category} = 'Reseller'"
	what_locked = false
	who = [
		{
			"user": "terraform@raito.io"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("raito_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("raito_filter.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("raito_filter.test", "table"),
						resource.TestCheckResourceAttr("raito_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("raito_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("raito_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("raito_filter.test", "what_locked", "false"),
					),
				},
			},
		})
	})
}
