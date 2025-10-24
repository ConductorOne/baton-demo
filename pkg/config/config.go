package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	GroupCountField = field.IntField("groups",
		field.WithDescription("Number of groups to create."),
		field.WithDefaultValue(3),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)
	ProjectCountField = field.IntField("projects",
		field.WithDescription("Number of projects to create."),
		field.WithDefaultValue(3),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)
	RoleCountField = field.IntField("roles",
		field.WithDescription("Number of roles to create."),
		field.WithDefaultValue(3),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)
	UserCountField = field.IntField("users",
		field.WithDescription("Number of users to create."),
		field.WithDefaultValue(10),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)
	InitDB                  = field.BoolField("init-db", field.WithDescription("Whether to initialize the database."), field.WithDefaultValue(false))
	DbFileName              = field.StringField("db-file-name", field.WithDescription("The name of the database file."), field.WithDefaultValue("baton-demo.db"))
	ResourceDropProbability = field.IntField(
		"resource-drop-probability",
		field.WithDescription("The probability we should drop any given resource from a sync, int from 0 - 100"),
		field.WithDefaultValue(0),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
		field.WithInt(func(r *field.IntRuler) { r.Lt(101) }),
	)
	RTDropProbability = field.IntField(
		"resource-type-drop-probability",
		field.WithDescription("The probability we should drop any whole resource type from the sync, int from 0 - 100"),
		field.WithDefaultValue(0),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
		field.WithInt(func(r *field.IntRuler) { r.Lt(101) }),
	)
	GrantDropProbability = field.IntField(
		"grant-drop-probability",
		field.WithDescription("The probability we should drop any grant from the sync, int from 0 - 100"),
		field.WithDefaultValue(0),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
		field.WithInt(func(r *field.IntRuler) { r.Lt(101) }),
	)
)

var relationships = []field.SchemaFieldRelationship{}

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	GroupCountField,
	ProjectCountField,
	RoleCountField,
	UserCountField,
	InitDB,
	DbFileName,
	ResourceDropProbability,
	RTDropProbability,
	GrantDropProbability,
}, field.WithConstraints(relationships...))
