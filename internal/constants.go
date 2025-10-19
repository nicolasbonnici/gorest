package internal

// Field names that should be excluded from Create/Update DTOs
const (
	FieldID        = "id"
	FieldCreatedAt = "created_at"
	FieldUpdatedAt = "updated_at"
)

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
