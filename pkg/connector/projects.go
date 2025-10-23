package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
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

func projectResource(p *client.Project, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	return sdkResource.NewResource(p.Name, projectResourceType, p.Id, sdkResource.WithParentResourceID(parentResourceID))
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
		project, err := projectResource(p, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		if !o.client.ShouldDrop() {
			ret = append(ret, project)
		}
	}

	return ret, "", nil, nil
}

func (o *projectBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	project, err := o.client.GetProject(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}

	resource, err := projectResource(project, parentResourceId)
	if err != nil {
		return nil, nil, err
	}

	return resource, nil, nil
}

// Entitlements returns two entitlements:
//   - Ownership of the project, grantable to a user
//   - Access to the project, grantable to groups
func (o *projectBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	access := sdkEntitlement.NewAssignmentEntitlement(resource, projectAccessEntitlement, sdkEntitlement.WithGrantableTo(groupResourceType, userResourceType))
	access.Description = fmt.Sprintf("Has access to the %s project", resource.DisplayName)

	owner := sdkEntitlement.NewPermissionEntitlement(resource, projectOwnerEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	owner.Description = fmt.Sprintf("Is the owner of the %s project", resource.DisplayName)

	return []*v2.Entitlement{access, owner}, "", nil, nil
}

// Grants returns grants for the access and owner entitlements. Only groups can be assigned to projects, but we will materialize group members as having access to the project.
func (o *projectBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	project, err := o.client.GetProject(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	offset := 0
	limit := 1000
	if pToken != nil {
		if pToken.Token != "" {
			offset, err = strconv.Atoi(pToken.Token)
			if err != nil {
				return nil, "", nil, err
			}
		}
		if pToken.Size > 0 {
			limit = pToken.Size
		}
	}
	var ret []*v2.Grant

	// Grant the owner entitlement to the project owner
	ownerID, err := sdkResource.NewResourceID(userResourceType, project.Owner)
	if err != nil {
		return nil, "", nil, err
	}

	if project.Owner != "" && offset == 0 {
		ret = append(ret, sdkGrant.NewGrant(resource, projectOwnerEntitlement, ownerID))
		// Owners also receive the access entitlement
		ret = append(ret, sdkGrant.NewGrant(resource, projectAccessEntitlement, ownerID))
	}

	// Iterate group assignments
	if len(project.GroupAssignments) > offset {
		end := min(offset+limit, len(project.GroupAssignments))
		for _, grpID := range project.GroupAssignments[offset:end] {
			pID, err := sdkResource.NewResourceID(groupResourceType, grpID)
			if err != nil {
				return nil, "", nil, err
			}

			entitlementIDs := []string{
				fmt.Sprintf("group:%s:member", grpID),
				fmt.Sprintf("group:%s:admin", grpID),
			}
			grant := sdkGrant.NewGrant(resource, projectAccessEntitlement, pID, sdkGrant.WithAnnotation(&v2.GrantExpandable{
				EntitlementIds: entitlementIDs,
			}))
			ret = append(ret, grant)
		}
	}

	nextPageToken := ""
	if len(project.GroupAssignments) > offset+limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}
	return ret, nextPageToken, nil, nil
}

func newProjectBuilder(client *client.Client) *projectBuilder {
	return &projectBuilder{
		client: client,
	}
}
