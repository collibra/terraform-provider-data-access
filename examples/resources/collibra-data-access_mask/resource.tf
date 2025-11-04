resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_mask" "example" {
  name        = "A Mask"
  description = "A simple mask"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
  ]
  type        = "SHA256"
  data_source = collibra-data-access_datasource.ds.id
  columns = [
    "SOME_DB.SOME_SCHEMA.SOME_TABLE.SOME_COLUMN"
  ]
}
