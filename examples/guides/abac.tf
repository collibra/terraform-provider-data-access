locals {
  example_grant_abac_rule = jsonencode(
    {
      aggregator : {
        operator : "Or"
        operands : [
          {
            aggregator : {
              operator : "And",
              operands : [
                {
                  comparison : {
                    operator : "HasTag",
                    leftOperand : "department",
                    rightOperand : {
                      literal : {
                        string : "Finance"
                      }
                    }
                  }
                },
                {
                  comparison : {
                    operator : "HasTag",
                    leftOperand : "sensitivity",
                    rightOperand : {
                      string : "PII"
                    }
                  }
                }
              ]
            }
          }
        ]
      }
    }
  )
}

resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant" "example_grant" {
  name        = "Grant with abac"
  description = "Grant with what abac rule"
  state       = "Active"
  what_abac_rule = {
    rule = local.example_grant_abac_rule
  }
  data_source = raito_datasource.ds.id
}