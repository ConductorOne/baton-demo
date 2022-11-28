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

type projectBuilder struct {
	client *client.Client
}

func (o *projectBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return projectResourceType
}

// List returns all the projects from the database as resource objects
// Projects don't include any traits because they don't match the 'shape' of any well known types.
func (o *projectBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	projects, err := o.client.ListProjects(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, p := range projects {
		resourceID, err := sdk.NewResourceID(projectResourceType, parentResourceID, p.Id)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, &v2.Resource{
			Id:          resourceID,
			DisplayName: p.Name,
		})
	}

	return ret, "", nil, nil
}

// Entitlements returns two entitlements:
//   - Ownership of the project, grantable to a user
//   - Access to the project, grantable to groups
func (o *projectBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var ret []*v2.Entitlement

	// This entitlement represents being a member of the group, and it can be granted to Users.
	ret = append(ret, &v2.Entitlement{
		Id:          sdk.NewEntitlementID(resource, "access"),
		Resource:    resource,
		DisplayName: "Access",
		GrantableTo: []*v2.ResourceType{
			groupResourceType,
			// Even though only groups can be assigned to a project, we will be materializing user access to the project based on their group membership.
			// In the future, this will be automatically handled for us.
			userResourceType,
		},
		Description: fmt.Sprintf("Has access to the %s project", resource.DisplayName),
		Slug:        "access", // Slug is a short name for the entitlement. This is often the same as display name.
	})

	ret = append(ret, &v2.Entitlement{
		Id:          sdk.NewEntitlementID(resource, "owner"),
		Resource:    resource,
		DisplayName: "Owner",
		GrantableTo: []*v2.ResourceType{userResourceType},
		Description: fmt.Sprintf("Is the owner of the %s project", resource.DisplayName),
		Slug:        "access", // Slug is a short name for the entitlement. This is often the same as display name.
	})

	return ret, "", nil, nil
}

// Grants returns grants for the access and owner entitlements. Only groups can be assigned to projects, but we will materialize group members as having access to the project.
func (o *projectBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	project, err := o.client.GetProject(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	// Projects emit an owner entitlement along with an access entitlement.
	ownerEntitlement := &v2.Entitlement{
		Id:       sdk.NewEntitlementID(resource, "owner"),
		Resource: resource,
	}
	accessEntitlement := &v2.Entitlement{
		Id:       sdk.NewEntitlementID(resource, "access"),
		Resource: resource,
	}

	// Grant the owner entitlement to the project owner
	ownerID, err := sdk.NewResourceID(userResourceType, nil, project.Owner)
	if err != nil {
		return nil, "", nil, err
	}
	ownerPrincipal := &v2.Resource{
		Id: ownerID,
	}
	ret = append(ret, &v2.Grant{
		Id:          sdk.NewGrantID(ownerEntitlement, ownerPrincipal),
		Entitlement: ownerEntitlement,
		Principal:   ownerPrincipal,
	})

	// Iterate group assignments
	for _, grpID := range project.GroupAssignments {
		pID, err := sdk.NewResourceID(groupResourceType, nil, grpID)
		if err != nil {
			return nil, "", nil, err
		}
		principal := &v2.Resource{
			Id: pID,
		}

		ret = append(ret, &v2.Grant{
			Id:          sdk.NewGrantID(accessEntitlement, principal),
			Entitlement: accessEntitlement,
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
				Id:          sdk.NewGrantID(accessEntitlement, adminPrincipal),
				Entitlement: accessEntitlement,
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
				Id:          sdk.NewGrantID(accessEntitlement, memberPrincipal),
				Entitlement: accessEntitlement,
				Principal:   memberPrincipal,
			})
		}

	}

	return ret, "", nil, nil
}

func newProjectBuilder(client *client.Client) *projectBuilder {
	return &projectBuilder{
		client: client,
	}
}
