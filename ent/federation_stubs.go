// Package ent contains federation stub entities for external services.
// These are placeholder types for entities owned by other services in the federation.
package ent

import (
	"github.com/google/uuid"
)

// User is a federation stub for the User entity owned by the auth service.
// This is not a real Ent entity - it's just a struct for federation resolution.
type User struct {
	ID uuid.UUID `json:"id"`
}

// IsEntity marks User as a federation entity
func (*User) IsEntity() {}

// UserDepartment is a federation stub for the UserDepartment entity owned by the auth service.
// This is not a real Ent entity - it's just a struct for federation resolution.
type UserDepartment struct {
	ID uuid.UUID `json:"id"`
}

// IsEntity marks UserDepartment as a federation entity
func (*UserDepartment) IsEntity() {}
