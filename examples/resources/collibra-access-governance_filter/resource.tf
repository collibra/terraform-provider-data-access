resource "collibra-access-governance_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-access-governance_purpose" "purpose1" {
  name        = "Purpose1"
  description = "Purpose"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    }
  ]
}

resource "collibra-access-governance_filter" "filter1" {
  name        = "First filter"
  description = "A simple filter"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
    {
      user : "user2@company.com"
      promise_duration : 604800
    },
    {
      access_control : collibra-access-governance_purpose.purpose1.id
    }
  ]
  data_source   = collibra-access-governance_datasource.ds.id
  table         = "database.schema.table"
  filter_policy = "{state} = 'NJ'"
}