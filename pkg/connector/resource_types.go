package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
}

var groupResourceType = &v2.ResourceType{
	Id:          "group",
	DisplayName: "Group",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
}

var roleResourceType = &v2.ResourceType{
	Id:          "role",
	DisplayName: "Role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
}

// Projects don't match any of the well-known resource traits
var projectResourceType = &v2.ResourceType{
	Id:          "project",
	DisplayName: "Project",
}
