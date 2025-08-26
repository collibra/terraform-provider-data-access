resource "raito_user" "u1" {
  name       = "user name"
  email      = "test-user@raito.io"
  raito_user = true
  type       = "Machine"
  password   = "!23vV678"
}