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
		roleTrait, err := sdk.NewRoleTrait(nil)
		if err != nil {
			return nil, "", nil, err
		}

		var annos annotations.Annotations
		annos.Append(roleTrait)

		resourceID, err := sdk.NewResourceID(roleResourceType, parentResourceID, r.Id)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, &v2.Resource{
			Id:          resourceID,
			DisplayName: r.Name,
			Annotations: annos,
		})
	}

	return ret, "", nil, nil
}

// Entitlements returns an assignment entitlement.
func (o *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var ret []*v2.Entitlement

	// This entitlement represents a User or Group being assigned the role
	ret = append(ret, &v2.Entitlement{
		Id:          sdk.NewEntitlementID(resource, "assignment"),
		Resource:    resource,
		DisplayName: "Assigned",
		GrantableTo: []*v2.ResourceType{
			userResourceType,  // Users can be directly assigned to the role
			groupResourceType, // Groups can also be assigned to the role
		},
		Description: fmt.Sprintf("Is assigned the %s role", resource.DisplayName),
		Slug:        "assigned", // Slug is a short name for the entitlement. This is often the same as display name.
	})

	return ret, "", nil, nil
}

// Grants returns grants for the assigned entitlement. We will return a grant for each group that is assigned the role, in addition to a grant for every member of the group/
// Users can also be directly assigned to a role to receive a grant.
func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	role, err := o.client.GetRole(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	// Roles emit an assignment entitlement for themselves.
	assignmentEntitlement := &v2.Entitlement{
		Id:       sdk.NewEntitlementID(resource, "assignment"),
		Resource: resource,
	}

	// Iterate direct assignments
	for _, userID := range role.DirectAssignments {
		pID, err := sdk.NewResourceID(userResourceType, nil, userID)
		if err != nil {
			return nil, "", nil, err
		}
		principal := &v2.Resource{
			Id: pID,
		}

		ret = append(ret, &v2.Grant{
			Id:          sdk.NewGrantID(assignmentEntitlement, principal),
			Entitlement: assignmentEntitlement,
			Principal:   principal,
		})
	}

	// Iterate group assignments
	for _, grpID := range role.GroupAssignments {
		pID, err := sdk.NewResourceID(groupResourceType, nil, grpID)
		if err != nil {
			return nil, "", nil, err
		}
		principal := &v2.Resource{
			Id: pID,
		}

		ret = append(ret, &v2.Grant{
			Id:          sdk.NewGrantID(assignmentEntitlement, principal),
			Entitlement: assignmentEntitlement,
			Principal:   principal,
		})

		// Look up group and iterate its members
		grp, err := o.client.GetGroup(ctx, grpID)
		if err != nil {
			return nil, "", nil, err
		}

		for _, adminID := range grp.Admins {
			adminPrincipalID, err := sdk.NewResourceID(userResourceType, nil, adminID)
			if err != nil {
				return nil, "", nil, err
			}
			adminPrincipal := &v2.Resource{
				Id: adminPrincipalID,
			}

			ret = append(ret, &v2.Grant{
				Id:          sdk.NewGrantID(assignmentEntitlement, adminPrincipal),
				Entitlement: assignmentEntitlement,
				Principal:   adminPrincipal,
			})
		}

		for _, memberID := range grp.Members {
			memberPrincipalID, err := sdk.NewResourceID(userResourceType, nil, memberID)
			if err != nil {
				return nil, "", nil, err
			}
			memberPrincipal := &v2.Resource{
				Id: memberPrincipalID,
			}

			ret = append(ret, &v2.Grant{
				Id:          sdk.NewGrantID(assignmentEntitlement, memberPrincipal),
				Entitlement: assignmentEntitlement,
				Principal:   memberPrincipal,
			})
		}
	}

	return ret, "", nil, nil
}

func newRoleBuilder(client *client.Client) *roleBuilder {
	return &roleBuilder{
		client: client,
	}
}
