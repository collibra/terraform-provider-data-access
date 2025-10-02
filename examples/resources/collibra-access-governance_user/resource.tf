resource "collibra-access-governance_user" "u1" {
  name          = "user name"
  email         = "test-user@collibra.com"
  collibra_user = true
  type          = "Machine"
  password      = "!23vV678"
}