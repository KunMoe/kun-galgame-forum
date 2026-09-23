package service

import (
	"errors"
	"fmt"

	"kun-galgame-api/pkg/perm"
)

// Operator is the caller of a permission write. Holds is the caller's own
// capability check, so a Bearer request can never pass a possession rule.
type Operator struct {
	ID    int
	Rank  int
	Holds func(perm.Permission) bool
}

type RoleLayer struct {
	Role      string
	Baseline  []perm.Permission
	Overrides []perm.Override
	Effective []perm.Permission
	Locked    bool
}

type Matrix struct {
	Catalog []perm.Permission
	Roles   []RoleLayer
}

type RoleChange struct {
	Role      string
	Overrides []perm.Override
}

type UserLayer struct {
	UserID    int
	Roles     []string
	Baseline  []perm.Permission
	Overrides []perm.Override
	Effective []perm.Permission
	Locked    bool
}

// Violation is one refused position in a permission write: Pointer is a JSON
// pointer into the request body, Parameter a path parameter name.
type Violation struct {
	Pointer   string
	Parameter string
	Reason    string
	Detail    string
}

type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("permission write refused at %d positions", len(e.Violations))
}

var (
	ErrUserNotFound = errors.New("permission: target user not found")
	ErrUserLookup   = errors.New("permission: target user lookup failed")
)

var matrixRoles = []string{"creator", "moderator", "admin", "ren"}

const roleRen = "ren"
