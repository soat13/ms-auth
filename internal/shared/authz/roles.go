package authz

import "errors"

var (
	ErrRolesRequired = errors.New("at least one role is required")
	ErrInvalidRole   = errors.New("invalid role")
)

var validRoles = map[string]bool{
	"USER":      true,
	"ADMIN":     true,
	"attendant": true,
	"manager":   true,
	"mechanic":  true,
}

// Roles represents a set of authorization roles assigned to a user.
type Roles []string

// Strings returns the roles as a plain string slice.
func (r Roles) Strings() []string {
	return []string(r)
}

// NewRoles constructs a Roles value from a plain string slice, validating them.
func NewRoles(roles []string) (Roles, error) {
	if len(roles) == 0 {
		return nil, ErrRolesRequired
	}

	for _, r := range roles {
		if !validRoles[r] {
			return nil, ErrInvalidRole
		}
	}

	return Roles(roles), nil
}

// NewRolesUnchecked constructs a Roles value without returning an error.
// Useful where the caller guarantees valid roles (like mapping from DB in auth adapter).
func NewRolesUnchecked(roles []string) Roles {
	return Roles(roles)
}
