resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_purpose" "purpose1" {
  name        = "Purpose1"
  description = "Purpose"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    }
  ]
}

resource "collibra-data-access_filter_rule" "filter1rule1" {
  name        = "table1 - NJ"
  description = "State = NJ for TABLE1"
  who = [
    {
      user : "user1@company.com"
    },
    {
      user : "user2@company.com"
      promise_duration : 604800
    },
    {
      access_control : collibra-data-access_purpose.purpose1.id
    }
  ]
  filter_policy = "{state} = 'NJ'"
  data_source   = data.collibra-data-access_datasource.ds.id
}

resource "collibra-data-access_filter_rule" "filter1rule2" {
  name        = "table1 - CA"
  description = "State = CA for TABLE1"
  who = [
    {
      user : "user3@company.com"
    },
  ]
  filter_policy = "{state} = 'CA'"
  data_source   = data.collibra-data-access_datasource.ds.id
}

resource "collibra-data-access_filter" "filter1" {
  name        = "First filter"
  description = "A simple filter"
  state       = "Active"

  table = {
    type        = "table"
    path        = ["DB1", "SCHEMA1", "TABLE1"]
    data_source = data.collibra-data-access_datasource.ds.id
  }
  filter_rules = [collibra-data-access_filter_rule.filter1rule1.id, collibra-data-access_filter_rule.filter1rule2.id]
}