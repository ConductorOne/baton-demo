package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var (
	groupMemberEntitlement = "member"
	groupAdminEntitlement  = "admin"
)

type groupBuilder struct {
	client *client.Client
}

func (o *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return groupResourceType
}

// List returns all the groups from the database as resource objects.
// Groups include the GroupTrait because they have the 'shape' of the well known Group type.
func (o *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	groups, err := o.client.ListGroups(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, g := range groups {
		// Group traits can contain arbitrary profile data
		profile := make(map[string]interface{})
		profile["group_color"] = "green"

		group, err := sdkResource.NewGroupResource(
			g.Name,
			groupResourceType,
			g.Id,
			[]sdkResource.GroupTraitOption{sdkResource.WithGroupProfile(profile)},
			sdkResource.WithParentResourceID(parentResourceID),
		)
		if err != nil {
			return nil, "", nil, err
		}
		ret = append(ret, group)
	}

	return ret, "", nil, nil
}

// Entitlements returns a membership and admin entitlement.
func (o *groupBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// This entitlement represents being a member of the group, and it can be granted to Users.
	member := sdkEntitlement.NewAssignmentEntitlement(resource, groupMemberEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	member.Description = fmt.Sprintf("Is a member of the %s group", resource.DisplayName)

	admin := sdkEntitlement.NewPermissionEntitlement(resource, groupAdminEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	admin.Description = fmt.Sprintf("Is an admin of the %s group", resource.DisplayName)

	return []*v2.Entitlement{member, admin}, "", nil, nil
}

// Grants returns grant information for group administrators and members.
func (o *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	grp, err := o.client.GetGroup(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	for _, adminID := range grp.Admins {
		pID, err := sdkResource.NewResourceID(userResourceType, adminID)
		if err != nil {
			return nil, "", nil, err
		}

		// Each admin gets the admin entitlement in addition to the member entitlement
		ret = append(ret, sdkGrant.NewGrant(resource, groupAdminEntitlement, pID))
		ret = append(ret, sdkGrant.NewGrant(resource, groupMemberEntitlement, pID))
	}

	for _, memberID := range grp.Members {
		pID, err := sdkResource.NewResourceID(userResourceType, memberID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, sdkGrant.NewGrant(resource, groupMemberEntitlement, pID))
	}

	return ret, "", nil, nil
}

func parseGroupID(groupID string) (string, string, error) {
	parts := strings.Split(groupID, ":")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid group ID %s", groupID)
	}

	return parts[1], parts[2], nil
}

func (o *groupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	if principal.Id.ResourceType != userResourceType.Id {
		return nil, nil, fmt.Errorf("baton-postgres: only users can have group memberships granted")
	}

	if entitlement.Resource.Id.ResourceType != groupResourceType.Id {
		return nil, nil, fmt.Errorf("baton-postgres: only groups can have memberships granted")
	}

	groupId, grantType, err := parseGroupID(entitlement.Id)
	if err != nil {
		return nil, nil, err
	}
	userID := principal.Id.Resource

	switch grantType {
	case "member":
		err := o.client.GrantGroupMember(ctx, groupId, userID)
		if err != nil {
			return nil, nil, err
		}
	case "admin":
		err := o.client.GrantGroupAdmin(ctx, groupId, userID)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf("baton-demo: unknown resource type")
	}

	return nil, nil, nil
}

func (o *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	role := grant.Entitlement.Resource.Id.Resource
	userID := grant.Principal.Id.Resource

	switch grant.Entitlement.Resource.Id.ResourceType {
	case roleResourceType.Id:
		err := o.client.RevokeRole(ctx, userID, role)
		if err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("baton-demo: unknown resource type")
	}
}

func newGroupBuilder(client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
