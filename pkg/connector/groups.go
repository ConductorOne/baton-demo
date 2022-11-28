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

type groupBuilder struct {
	client *client.Client
}

func (o *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return groupResourceType
}

// List returns all the groups from the database as resource objects
// Groups include the GroupTrait because they have the 'shape' of the well known Group type
func (o *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	groups, err := o.client.ListGroups(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, g := range groups {
		// Group traits can contain an asset reference to an icon for the group, as well as arbitrary profile data
		profile := make(map[string]interface{})
		profile["group_color"] = "green"

		groupTrait, err := sdk.NewGroupTrait(nil, profile)
		if err != nil {
			return nil, "", nil, err
		}

		var annos annotations.Annotations
		annos.Append(groupTrait)

		resourceID, err := sdk.NewResourceID(groupResourceType, parentResourceID, g.Id)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, &v2.Resource{
			Id:          resourceID,
			DisplayName: g.Name,
			Annotations: annos,
		})
	}

	return ret, "", nil, nil
}

// Entitlements returns a membership and admin entitlement
func (o *groupBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var ret []*v2.Entitlement

	// This entitlement represents being a member of the group, and it can be granted to Users.
	ret = append(ret, &v2.Entitlement{
		Id:          sdk.NewEntitlementID(resource, "member"),
		Resource:    resource,
		DisplayName: "Member",
		GrantableTo: []*v2.ResourceType{userResourceType},
		Description: fmt.Sprintf("Is a member of the %s group", resource.DisplayName),
		Slug:        "member", // Slug is a short name for the entitlement. This is often the same as display name.
	})

	// This entitlement represents being an admin of the group, and it can be granted to Users.
	ret = append(ret, &v2.Entitlement{
		Id:          sdk.NewEntitlementID(resource, "admin"),
		Resource:    resource,
		DisplayName: "Administrator",
		GrantableTo: []*v2.ResourceType{userResourceType},
		Description: fmt.Sprintf("Is an admin of the %s group", resource.DisplayName),
		Slug:        "admin", // Slug is a short name for the entitlement. This is often the same as display name.
	})

	return ret, "", nil, nil
}

// Grants returns grant information for group administrators and members
func (o *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	grp, err := o.client.GetGroup(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	// Groups emit membership and memberID entitlements. We need to calculate grants for those.
	// Admin grants
	adminEntitlement := &v2.Entitlement{
		Id:       sdk.NewEntitlementID(resource, "admin"),
		Resource: resource,
	}
	for _, adminID := range grp.Admins {
		pID, err := sdk.NewResourceID(userResourceType, nil, adminID)
		if err != nil {
			return nil, "", nil, err
		}
		principal := &v2.Resource{
			Id: pID,
		}

		ret = append(ret, &v2.Grant{
			Id:          sdk.NewGrantID(adminEntitlement, principal),
			Entitlement: adminEntitlement,
			Principal:   principal,
		})
	}

	// Member grants
	memberEntitlement := &v2.Entitlement{
		Id:       sdk.NewEntitlementID(resource, "member"),
		Resource: resource,
	}

	for _, memberID := range grp.Members {
		pID, err := sdk.NewResourceID(userResourceType, nil, memberID)
		if err != nil {
			return nil, "", nil, err
		}
		principal := &v2.Resource{
			Id: pID,
		}

		ret = append(ret, &v2.Grant{
			Id:          sdk.NewGrantID(memberEntitlement, principal),
			Entitlement: memberEntitlement,
			Principal:   principal,
		})
	}

	return ret, "", nil, nil
}

func newGroupBuilder(client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
