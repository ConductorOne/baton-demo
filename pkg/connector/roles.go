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
	roleAssignmentEntitlement = "assignment"
)

type roleBuilder struct {
	client *client.Client
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

// List returns all the roles from the database as resource objects
// Roles include the role trait because they have the 'shape' of the well known Role type.
func (o *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	roles, err := o.client.ListRoles(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, r := range roles {
		role, err := sdkResource.NewRoleResource(r.Name, roleResourceType, r.Id, nil, sdkResource.WithParentResourceID(parentResourceID))
		if err != nil {
			return nil, "", nil, err
		}
		ret = append(ret, role)
	}

	return ret, "", nil, nil
}

// Entitlements returns an assignment entitlement.
func (o *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// This entitlement represents a User or Group being assigned the role
	assignment := sdkEntitlement.NewAssignmentEntitlement(resource, roleAssignmentEntitlement, sdkEntitlement.WithGrantableTo(userResourceType, groupResourceType))
	assignment.Description = fmt.Sprintf("Is assigned the %s role", resource.DisplayName)

	return []*v2.Entitlement{assignment}, "", nil, nil
}

// Grants returns grants for the assigned entitlement. We will return a grant for each group that is assigned the role, in addition to a grant for every member of the group/
// Users can also be directly assigned to a role to receive a grant.
func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	role, err := o.client.GetRole(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	// Iterate direct assignments
	for _, userID := range role.DirectAssignments {
		pID, err := sdkResource.NewResourceID(userResourceType, userID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, sdkGrant.NewGrant(resource, roleAssignmentEntitlement, pID))
	}

	// Iterate group assignments
	for _, grpID := range role.GroupAssignments {
		pID, err := sdkResource.NewResourceID(groupResourceType, grpID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, sdkGrant.NewGrant(resource, roleAssignmentEntitlement, pID))

		// Look up group and iterate its members
		grp, err := o.client.GetGroup(ctx, grpID)
		if err != nil {
			return nil, "", nil, err
		}

		// Grant all admins and members the assignment entitlement
		for _, userID := range append(grp.Admins, grp.Members...) {
			// FIXME(morgabra): What should we do here?
			if strings.HasPrefix(userID, "group:") {
				continue
			}
			pID, err := sdkResource.NewResourceID(userResourceType, userID)
			if err != nil {
				return nil, "", nil, err
			}

			ret = append(ret, sdkGrant.NewGrant(resource, roleAssignmentEntitlement, pID))
		}
	}

	return ret, "", nil, nil
}

func newRoleBuilder(client *client.Client) *roleBuilder {
	return &roleBuilder{
		client: client,
	}
}
