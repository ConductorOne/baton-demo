package connector

import (
	"context"
	"fmt"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"

	"github.com/conductorone/baton-demo/pkg/client"
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

func (o *groupBuilder) entitlements(ctx context.Context, resource *v2.Resource) (*v2.Entitlement, *v2.Entitlement) {
	// This entitlement represents being a member of the group, and it can be granted to Users.
	member := sdkEntitlement.NewAssignmentEntitlement(resource, groupMemberEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	member.Description = fmt.Sprintf("Is a member of the %s group", resource.DisplayName)

	admin := sdkEntitlement.NewPermissionEntitlement(resource, groupAdminEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	admin.Description = fmt.Sprintf("Is an admin of the %s group", resource.DisplayName)

	return member, admin
}

// Entitlements returns a membership and admin entitlement.
func (o *groupBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// This entitlement represents being a member of the group, and it can be granted to Users.
	member, admin := o.entitlements(ctx, resource)
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
		var pID *v2.ResourceId
		if strings.HasPrefix(memberID, "group:") {
			memberID = strings.TrimPrefix(memberID, "group:")
			pID, err = sdkResource.NewResourceID(groupResourceType, memberID)
			if err != nil {
				return nil, "", nil, err
			}

			g := sdkGrant.NewGrant(resource, groupMemberEntitlement, pID)

			// FIXME(morgabra): Make a helper/refactor this:
			member, admin := o.entitlements(ctx, &v2.Resource{Id: &v2.ResourceId{
				ResourceType: groupResourceType.Id,
				Resource:     memberID,
			}})
			annos := annotations.Annotations(g.Annotations)
			annos.Append(&v2.GrantExpandable{
				EntitlementIds: []string{member.Id, admin.Id},
			})
			g.Annotations = annos
			ret = append(ret, g)
		} else {
			pID, err = sdkResource.NewResourceID(userResourceType, memberID)
			if err != nil {
				return nil, "", nil, err
			}
			ret = append(ret, sdkGrant.NewGrant(resource, groupMemberEntitlement, pID))
		}

	}

	return ret, "", nil, nil
}

func newGroupBuilder(client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
