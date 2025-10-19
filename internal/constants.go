package internal

// Field names that should be excluded from Create/Update DTOs
const (
	FieldID        = "id"
	FieldCreatedAt = "created_at"
	FieldUpdatedAt = "updated_at"
)

// Sensitive field patterns that should be excluded from Response DTOs
var SensitiveFieldPatterns = []string{
	"password",
	"hashed_password",
	"password_hash",
	"secret",
	"api_key",
	"token",
	"refresh_token",
	"access_token",
}

// Table names
const (
	UsersTable = "users"
)

// Pagination constants
const (
	DefaultPageSize = 20
	MaxPageSize     = 1000
	MinPageSize     = 1
)
