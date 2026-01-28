package connector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var (
	scopedRoleAssignmentEntitlement = "assignment"
)

type scopedRoleBuilder struct {
	client *client.Client
}

func (o *scopedRoleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return scopedRoleResourceType
}

func makeScopedRoleResourceID(roleID, projectID string) string {
	return fmt.Sprintf("%s:%s", roleID, projectID)
}

func parseScopedRoleResourceID(id string) (string, string, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid scoped role resource id")
	}
	return parts[0], parts[1], nil
}

func (o *scopedRoleBuilder) scopedRoleResource(ctx context.Context, r *client.ScopedRole, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	role, err := o.client.GetRole(ctx, r.RoleId)
	if err != nil {
		return nil, err
	}
	project, err := o.client.GetProject(ctx, r.ProjectId)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("Scoped Role %s on Project %s", role.Name, project.Name)
	roleResource, err := roleResource(role, nil)
	if err != nil {
		return nil, err
	}
	projectResource, err := projectResource(project, nil)
	if err != nil {
		return nil, err
	}
	traits := []rs.ScopeBindingTraitOption{
		rs.WithRoleScopeRoleId(roleResource.Id),
		rs.WithRoleScopeResourceId(projectResource.Id),
	}
	id := makeScopedRoleResourceID(r.RoleId, r.ProjectId)
	return rs.NewScopeBindingResource(name, scopedRoleResourceType, id, traits, rs.WithParentResourceID(parentResourceID))
}

// List returns all the roles from the database as resource objects
// Roles include the role trait because they have the 'shape' of the well known Role type.
func (o *scopedRoleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	scopedRoles, err := o.client.ListScopedRoles(ctx)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("Scoped roles: %+v\n", scopedRoles)

	var ret []*v2.Resource
	for _, sr := range scopedRoles {
		scopedRole, err := o.scopedRoleResource(ctx, sr, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		ret = append(ret, scopedRole)
	}

	return ret, nil, nil
}

func (o *scopedRoleBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	roleID, projectID, err := parseScopedRoleResourceID(resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	scopedRole, err := o.client.GetScopedRole(ctx, roleID, projectID)
	if err != nil {
		return nil, nil, err
	}
	resource, err := o.scopedRoleResource(ctx, scopedRole, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return resource, nil, nil
}

// Entitlements returns an assignment entitlement.
func (o *scopedRoleBuilder) Entitlements(_ context.Context, resource *v2.Resource, ops rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	// This entitlement represents a User or Group being assigned the role
	assignment := sdkEntitlement.NewAssignmentEntitlement(
		resource,
		scopedRoleAssignmentEntitlement,
		sdkEntitlement.WithGrantableTo(userResourceType),
	)
	assignment.Description = fmt.Sprintf("Is assigned the %s role", resource.DisplayName)

	return []*v2.Entitlement{assignment}, nil, nil
}

func (o *scopedRoleBuilder) Grants(ctx context.Context, r *v2.Resource, ops rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	roleID, projectID, err := parseScopedRoleResourceID(r.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	scopedRole, err := o.client.GetScopedRole(ctx, roleID, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// TODO: Pagination.
	fmt.Printf("Scoped role: %+v\n", scopedRole)

	grants := []*v2.Grant{}
	// Get all grants for the role.
	for _, userID := range scopedRole.UserAssignments {
		pID, err := rs.NewResourceID(userResourceType, userID)
		if err != nil {
			return nil, nil, err
		}
		grants = append(grants, sdkGrant.NewGrant(r, scopedRoleAssignmentEntitlement, pID))
	}
	return grants, nil, nil
}

func newScopedRoleBuilder(client *client.Client) *scopedRoleBuilder {
	return &scopedRoleBuilder{
		client: client,
	}
}
