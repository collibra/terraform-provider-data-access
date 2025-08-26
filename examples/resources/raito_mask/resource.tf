resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_mask" "example" {
  name        = "A Raito Mask"
  description = "A simple mask"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
  ]
  type        = "SHA256"
  data_source = raito_datasource.ds.id
  columns = [
    "SOME_DB.SOME_SCHEMA.SOME_TABLE.SOME_COLUMN"
  ]
}