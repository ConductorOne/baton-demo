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
	ScopedRoleCountField = field.IntField("scoped-roles",
		field.WithDescription("Number of scoped roles to create."),
		field.WithDefaultValue(3),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)
	UserCountField = field.IntField("users",
		field.WithDescription("Number of human users to create."),
		field.WithDefaultValue(10),
		field.WithInt(func(r *field.IntRuler) { r.Gt(0) }),
	)

	// NHI estate fields. These let an operator stand up a realistic
	// non-human-identity demo estate: service/system accounts (K2), secrets
	// bound to them (K1), app/role non-human identities (K3), and agents that
	// authenticate as a service account.
	ServiceAccountCountField = field.IntField("service-accounts",
		field.WithDescription("Number of service-account users (account_type SERVICE) to create."),
		field.WithDefaultValue(4),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	SystemAccountCountField = field.IntField("system-accounts",
		field.WithDescription("Number of system users (account_type SYSTEM) to create."),
		field.WithDefaultValue(1),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	SecretCountField = field.IntField("secrets",
		field.WithDescription("Number of secrets (TRAIT_SECRET) bound to service accounts via identity_id."),
		field.WithDefaultValue(6),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	UnownedSecretCountField = field.IntField("unowned-secrets",
		field.WithDescription("Number of secrets with no owning identity (no identity_id back-ref)."),
		field.WithDefaultValue(2),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	NHIAppCountField = field.IntField("nhi-apps",
		field.WithDescription("Number of non-human-identity apps (TRAIT_APP + NonHumanIdentityTrait, app-registration/managed-identity)."),
		field.WithDefaultValue(3),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	AssumableRoleCountField = field.IntField("assumable-roles",
		field.WithDescription("Number of assumable-role non-human identities (TRAIT_ROLE + NonHumanIdentityTrait)."),
		field.WithDefaultValue(2),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)
	AgentCountField = field.IntField("agents",
		field.WithDescription("Number of agents (TRAIT_AGENT) each backed by a service account."),
		field.WithDefaultValue(2),
		field.WithInt(func(r *field.IntRuler) { r.Gte(0) }),
	)

	InitDB     = field.BoolField("init-db", field.WithDescription("Whether to initialize the database."), field.WithDefaultValue(false))
	DbFileName = field.StringField("db-file-name", field.WithDescription("The name of the database file."), field.WithDefaultValue("baton-demo.db"))
)

var relationships = []field.SchemaFieldRelationship{}

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	GroupCountField,
	ProjectCountField,
	RoleCountField,
	ScopedRoleCountField,
	UserCountField,
	ServiceAccountCountField,
	SystemAccountCountField,
	SecretCountField,
	UnownedSecretCountField,
	NHIAppCountField,
	AssumableRoleCountField,
	AgentCountField,
	InitDB,
	DbFileName,
}, field.WithConstraints(relationships...))
