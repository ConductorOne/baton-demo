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
	InitDB          = field.BoolField("init-db", field.WithDescription("Whether to initialize the database."), field.WithDefaultValue(false))
	DbFileName      = field.StringField("db-file-name", field.WithDescription("The name of the database file."), field.WithDefaultValue("baton-demo.db"))
	DropProbability = field.IntField("drop-probability", field.WithDescription("The probability we should drop any given value from a sync, int from 0 - 100"), field.WithDefaultValue(0))
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
	DropProbability,
}, field.WithConstraints(relationships...))
