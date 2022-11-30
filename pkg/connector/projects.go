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
	projectOwnerEntitlement  = "owner"
	projectAccessEntitlement = "access"
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
		project, err := sdk.NewResource(p.Name, projectResourceType, parentResourceID, p.Id)
		if err != nil {
			return nil, "", nil, err
		}
		ret = append(ret, project)
	}

	return ret, "", nil, nil
}

// Entitlements returns two entitlements:
//   - Ownership of the project, grantable to a user
//   - Access to the project, grantable to groups
func (o *projectBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	access := sdk.NewAssignmentEntitlement(resource, projectAccessEntitlement, groupResourceType, userResourceType)
	access.Description = fmt.Sprintf("Has access to the %s project", resource.DisplayName)

	owner := sdk.NewPermissionEntitlement(resource, projectOwnerEntitlement, userResourceType)
	owner.Description = fmt.Sprintf("Is the owner of the %s project", resource.DisplayName)

	return []*v2.Entitlement{access, owner}, "", nil, nil
}

// Grants returns grants for the access and owner entitlements. Only groups can be assigned to projects, but we will materialize group members as having access to the project.
func (o *projectBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	project, err := o.client.GetProject(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Grant

	// Grant the owner entitlement to the project owner
	ownerID, err := sdk.NewResourceID(userResourceType, project.Owner)
	if err != nil {
		return nil, "", nil, err
	}

	ret = append(ret, sdk.NewGrant(resource, projectOwnerEntitlement, ownerID))
	// Owners also receive the access entitlement
	ret = append(ret, sdk.NewGrant(resource, projectAccessEntitlement, ownerID))

	// Iterate group assignments
	for _, grpID := range project.GroupAssignments {
		pID, err := sdk.NewResourceID(groupResourceType, grpID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, sdk.NewGrant(resource, projectAccessEntitlement, pID))

		// Look up group and iterate its members
		grp, err := o.client.GetGroup(ctx, grpID)
		if err != nil {
			return nil, "", nil, err
		}

		for _, userID := range append(grp.Admins, grp.Members...) {
			pID, err := sdk.NewResourceID(userResourceType, userID)
			if err != nil {
				return nil, "", nil, err
			}

			ret = append(ret, sdk.NewGrant(resource, projectAccessEntitlement, pID))
		}
	}

	return ret, "", nil, nil
}

func newProjectBuilder(client *client.Client) *projectBuilder {
	return &projectBuilder{
		client: client,
	}
}
