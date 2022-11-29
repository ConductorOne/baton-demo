package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/sdk"
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

		group, err := sdk.NewGroupResource(g.Name, groupResourceType, parentResourceID, g.Id, profile)
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
	member := sdk.NewAssignmentEntitlement(resource, groupMemberEntitlement, userResourceType)
	member.Description = fmt.Sprintf("Is a member of the %s group", resource.DisplayName)

	admin := sdk.NewPermissionEntitlement(resource, groupAdminEntitlement, userResourceType)
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
		pID, err := sdk.NewResourceID(userResourceType, nil, adminID)
		if err != nil {
			return nil, "", nil, err
		}

		// Each admin gets the admin entitlement in addition to the member entitlement
		ret = append(ret, sdk.NewGrant(resource, groupAdminEntitlement, pID))
		ret = append(ret, sdk.NewGrant(resource, groupMemberEntitlement, pID))
	}

	for _, memberID := range grp.Members {
		pID, err := sdk.NewResourceID(userResourceType, nil, memberID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, sdk.NewGrant(resource, groupMemberEntitlement, pID))
	}

	return ret, "", nil, nil
}

func newGroupBuilder(client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
