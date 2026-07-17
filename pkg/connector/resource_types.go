package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// The user resource type is for all user objects from the database.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
}

// The group resource type is for all group objects from the database.
var groupResourceType = &v2.ResourceType{
	Id:          "group",
	DisplayName: "Group",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
}

// The role resource type is for all role objects from the database.
var roleResourceType = &v2.ResourceType{
	Id:          "role",
	DisplayName: "Role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
}

// The scoped role resource type is for all scoped role objects from the database.
var scopedRoleResourceType = &v2.ResourceType{
	Id:          "scoped_role",
	DisplayName: "Scoped Role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_SCOPE_BINDING},
}

// The project resource type is for all project objects from the database
// Projects don't match any of the well-known resource traits.
var projectResourceType = &v2.ResourceType{
	Id:          "project",
	DisplayName: "Project",
}

// The secret resource type is for credentials (K1). Secrets carry a SecretTrait
// and are inventory-only, so they skip entitlements and grants.
var secretResourceType = &v2.ResourceType{
	Id:          "secret",
	DisplayName: "Secret",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_SECRET},
	Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
}

// The nhi_app resource type is for non-human-identity apps (K3): TRAIT_APP
// resources carrying a NonHumanIdentityTrait.
var nhiAppResourceType = &v2.ResourceType{
	Id:          "nhi_app",
	DisplayName: "NHI App",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_APP},
	Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
}

// The demo_app resource type is a single static App the last-login
// usage-event feed targets, mirroring baton-dropbox's static "Dropbox" app.
// UsageEvents attach to it; its "access" entitlement is what C1's usage uplift
// reads the resulting usage principals through.
var demoAppResourceType = &v2.ResourceType{
	Id:          "demo_app",
	DisplayName: "Demo App",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_APP},
}

// The assumable_role resource type is for non-human-identity assumable roles
// (K3): TRAIT_ROLE resources carrying a NonHumanIdentityTrait.
var assumableRoleResourceType = &v2.ResourceType{
	Id:          "assumable_role",
	DisplayName: "Assumable Role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
}

// The agent resource type is for autonomous non-human actors that authenticate
// as an identity: TRAIT_AGENT resources carrying an AgentTrait.
var agentResourceType = &v2.ResourceType{
	Id:          "agent",
	DisplayName: "Agent",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_AGENT},
	Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
}
