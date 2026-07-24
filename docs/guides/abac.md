---
page_title: "Abac Rules in Collibra Data Access"
---

# Abac Rules in the Collibra Data Access provider

Access provider resources (roles, column masks, and row filters) use dynamic rules to define who can access what. Specify these rules in JSON format using the following structure.

## JSON Structure

Dynamic rules consist of nested objects and arrays that represent logical expressions.

* **DynamicRule:**
    Define exactly one of the following expressions:

    * `literal`: A Boolean value representing a truth condition.
    * `comparison`: A `Comparison` expression with an operator, left operand, and right operand.
    * `aggregator`: An `Aggregator` expression combining multiple binary expressions using AND or OR logic.
    * `unaryExpression`: A single expression negated using a NOT operator.

* **Comparison:**
    * `operator`: The comparison type (for example, HasTag, ContainsTag, InheritsTag).
    * `leftOperand`: The tag key used in the comparison.
    * `rightOperand`: The value to compare against `Operand`.

* **Operand:**
    * `literal`: A `Literal` value, including Boolean, strings, or string lists.

* **Literal:**
    Define exactly one of the following expressions:

    * `bool`: A boolean value.
    * `string`: A string value.
    * `stringList`: A list of string values.

* **Aggregator:**
    * `operator`: The aggregation type (for example, `And` or `Or`).
    * `operands`: An array of `DynamicRule` objects.

* **Unary:**
    * `operator`: The unary type (for example, `Not`).
    * `operands`: An array of `DynamicRule` objects.

## Constraints

The following constraints are evaluated during the creation of a dynamic rule:

* The first level should be an aggregation with operator `Or`.
* The second level should be an aggregation with operator `And`.
* The third level can be `Comparison`, `Literal`, or `Unary`.
* If a unary expression is used at the third level, the fourth level should be `Comparison` or `Literal`.

## Example Rule

The following JSON rule represents a condition that evaluates to true if the tag `department` has the value `Finance` and the tag `sensitivity` has the value `PII`.

```json
{
  "aggregator": {
    "operator": "Or",
    "operands": [
      {
        "aggregator": {
          "operator": "And",
          "operands": [
            {
              "comparison": {
                "operator": "HasTag",
                "leftOperand": "department",
                "rightOperand": {
                  "literal": {
                    "string": "Finance"
                  }
                }
              }
            },
            {
              "comparison": {
                "operator": "HasTag",
                "leftOperand": "sensitivity",
                "rightOperand": {
                  "literal": {
                    "string": "PII"
                  }
                }
              }
            }
          ]
        }
      }
    ]
  }
} 
```

## Example in Terraform

```terraform
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

resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_grant" "example_grant" {
  name        = "Grant with abac"
  description = "Grant with what abac rule"
  state       = "Active"
  what_abac_rules = [
    {
      rule : local.example_grant_abac_rule
      id : "my_rule"
      do_types : ["table", "view"]
      permissions : ["SELECT"]
      scope : [
        {
          type : "database"
          path : ["my_database"]
          data_source : collibra-data-access_datasource.ds.id
        }
      ]
    }
  ]
  data_sources = [
    {
      data_source : collibra-data-access_datasource.ds.id
      type : "role"
    }
  ]
}
```
