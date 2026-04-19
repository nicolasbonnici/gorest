package rbac

import "time"

type Policy string

const (
	DenyAll  Policy = "deny_all"
	AllowAll Policy = "allow_all"
)

type Role struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Parent      string    `json:"parent,omitempty" db:"parent"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserRole struct {
	UserID     string    `json:"user_id" db:"user_id"`
	RoleID     string    `json:"role_id" db:"role_id"`
	AssignedAt time.Time `json:"assigned_at" db:"assigned_at"`
	AssignedBy string    `json:"assigned_by" db:"assigned_by"`
}

type Permission struct {
	Field string
	Read  []string
	Write []string
}

type PermissionSet map[string]Permission

type AuditEntry struct {
	ID        string    `json:"id" db:"id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	UserID    string    `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"` // "promote", "demote", "add_role", "remove_role"
	Role      string    `json:"role" db:"role"`
	Actor     string    `json:"actor" db:"actor"`
	Success   bool      `json:"success" db:"success"`
	Error     string    `json:"error,omitempty" db:"error"`
}

type UserRoles struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RoleNode struct {
	Name     string
	Children []*RoleNode
}

func (p Permission) HasReadPermission(userRoles []string) bool {
	for _, allowedRole := range p.Read {
		switch allowedRole {
		case "*":
			return true
		case "any":
			return len(userRoles) > 0
		case "none":
			continue
		default:
			for _, userRole := range userRoles {
				if userRole == allowedRole {
					return true
				}
			}
		}
	}
	return false
}

func (p Permission) HasWritePermission(userRoles []string) bool {
	for _, allowedRole := range p.Write {
		switch allowedRole {
		case "*":
			return true
		case "any":
			return len(userRoles) > 0
		case "none":
			continue
		default:
			for _, userRole := range userRoles {
				if userRole == allowedRole {
					return true
				}
			}
		}
	}
	return false
}
