package types

const (
	// Global permissions
	GlobalPermissionRead  = "READ"
	GlobalPermissionWrite = "WRITE"
	GlobalPermissionAdmin = "ADMIN"
)

var AllGlobalPermissions = []string{
	GlobalPermissionRead,
	GlobalPermissionWrite,
	GlobalPermissionAdmin,
}
