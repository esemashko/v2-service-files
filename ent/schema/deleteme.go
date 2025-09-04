package schema

import (
	localmixin "main/ent/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/privacy"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Tenant holds the schema definition for the Tenant entity
type Tenant struct {
	ent.Schema
}

// Mixin of the User
func (Tenant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		localmixin.TimeMixin{},
		localmixin.UserMixin{},
		localmixin.SoftDeleteMixin{},
		localmixin.LimitMixin{},
	}
}

// Policy defines the privacy policy using centralized user privacy rules
func (Tenant) Policy() ent.Policy {
	return privacy.Policy{}
}

func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
	}
}

func (Tenant) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Tenant
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{}
}

// Annotations with ACL rules
func (Tenant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "deleteme"},
	}
}
