package connector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	scopedRoleAssignmentEntitlement = "assignment"
)

type scopedRoleBuilder struct {
	client *client.Client
}

var _ connectorbuilder.ResourceSyncerV2 = (*scopedRoleBuilder)(nil)
var _ connectorbuilder.ResourceProvisionerV2Limited = (*scopedRoleBuilder)(nil)

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

func (o *scopedRoleBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	if principal.Id.ResourceType != userResourceType.Id {
		return nil, nil, status.Errorf(codes.InvalidArgument, "baton-demo: only users can have scoped roles granted")
	}

	// Get Scope binding trait from resource
	scopeTrait, err := rs.GetScopeBindingTrait(entitlement.Resource)
	if err != nil {
		return nil, nil, err
	}
	if scopeTrait == nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "scope binding trait was not found on resource")
	}

	roleID := scopeTrait.RoleId.Resource
	projectID := scopeTrait.ScopeResourceId.Resource

	// Make sure the role and project exist.
	_, err = o.client.GetRole(ctx, roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, status.Errorf(codes.NotFound, "baton-demo: role not found")
		}
		return nil, nil, err
	}
	_, err = o.client.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, status.Errorf(codes.NotFound, "baton-demo: project not found")
		}
		return nil, nil, err
	}

	// Create scoped role if it doesn't exist.
	scopedRole, err := o.client.GetScopedRole(ctx, roleID, projectID)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		scopedRole = &client.ScopedRole{
			Id:              makeScopedRoleResourceID(roleID, projectID),
			ProjectId:       projectID,
			RoleId:          roleID,
			UserAssignments: []string{principal.Id.Resource},
		}
		err = o.client.CreateScopedRole(ctx, scopedRole)
		if err != nil {
			return nil, nil, err
		}
		resource, err := o.scopedRoleResource(ctx, scopedRole, nil)
		if err != nil {
			return nil, nil, err
		}
		scopedRoleGrant := sdkGrant.NewGrant(resource, scopedRoleAssignmentEntitlement, principal.Id)
		return []*v2.Grant{scopedRoleGrant}, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	if slices.Contains(scopedRole.UserAssignments, principal.Id.Resource) {
		scopedRoleGrant := sdkGrant.NewGrant(entitlement.Resource, scopedRoleAssignmentEntitlement, principal.Id)
		return []*v2.Grant{scopedRoleGrant}, annotations.New(&v2.GrantAlreadyExists{}), nil
	}

	scopedRole.UserAssignments = append(scopedRole.UserAssignments, principal.Id.Resource)
	err = o.client.UpdateScopedRole(ctx, scopedRole)
	if err != nil {
		return nil, nil, err
	}
	resource, err := o.scopedRoleResource(ctx, scopedRole, nil)
	if err != nil {
		return nil, nil, err
	}
	scopedRoleGrant := sdkGrant.NewGrant(resource, scopedRoleAssignmentEntitlement, principal.Id)
	return []*v2.Grant{scopedRoleGrant}, nil, nil
}

func (o *scopedRoleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	principal := grant.Principal
	entitlement := grant.Entitlement

	if principal.Id.ResourceType != userResourceType.Id {
		return nil, status.Errorf(codes.InvalidArgument, "baton-demo: only users can have scoped roles revoked")
	}

	roleID, projectID, err := parseScopedRoleResourceID(entitlement.GetResource().GetId().GetResource())
	if err != nil {
		return nil, err
	}

	scopedRole, err := o.client.GetScopedRole(ctx, roleID, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return annotations.New(&v2.GrantAlreadyRevoked{}), nil
		}
		return nil, err
	}

	if !slices.Contains(scopedRole.UserAssignments, principal.Id.Resource) {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	scopedRole.UserAssignments = slices.DeleteFunc(scopedRole.UserAssignments, func(userID string) bool {
		return userID == principal.Id.Resource
	})
	err = o.client.UpdateScopedRole(ctx, scopedRole)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func newScopedRoleBuilder(client *client.Client) *scopedRoleBuilder {
	return &scopedRoleBuilder{
		client: client,
	}
}
