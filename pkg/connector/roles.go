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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func roleResource(r *client.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	return sdkResource.NewRoleResource(r.Name, roleResourceType, r.Id, nil, sdkResource.WithParentResourceID(parentResourceID))
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
		role, err := roleResource(r, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		ret = append(ret, role)
	}

	return ret, "", nil, nil
}

func (o *roleBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	role, err := o.client.GetRole(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}

	resource, err := roleResource(role, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return resource, nil, nil
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

	// Iterate direct assignments
	if len(role.DirectAssignments) > offset {
		end := min(offset+limit, len(role.DirectAssignments))
		for _, userID := range role.DirectAssignments[offset:end] {
			pID, err := sdkResource.NewResourceID(userResourceType, userID)
			if err != nil {
				return nil, "", nil, err
			}

			ret = append(ret, sdkGrant.NewGrant(resource, roleAssignmentEntitlement, pID))
		}
	}

	// Iterate group assignments
	if len(role.GroupAssignments) > offset {
		end := min(offset+limit, len(role.GroupAssignments))
		for _, grpID := range role.GroupAssignments[offset:end] {
			pID, err := sdkResource.NewResourceID(groupResourceType, grpID)
			if err != nil {
				return nil, "", nil, err
			}

			entitlementIDs := []string{
				fmt.Sprintf("group:%s:member", grpID),
				fmt.Sprintf("group:%s:admin", grpID),
			}
			grant := sdkGrant.NewGrant(resource, roleAssignmentEntitlement, pID, sdkGrant.WithAnnotation(&v2.GrantExpandable{
				EntitlementIds: entitlementIDs,
			}))
			ret = append(ret, grant)
		}
	}

	nextPageToken := ""
	if len(role.DirectAssignments) > offset+limit || len(role.GroupAssignments) > offset+limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return ret, nextPageToken, nil, nil
}

func (o *roleBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	if principal.Id.ResourceType != userResourceType.Id {
		return nil, nil, fmt.Errorf("baton-postgres: only users can have roles granted")
	}

	role := entitlement.Resource.Id.Resource
	userID := principal.Id.Resource

	switch entitlement.Resource.Id.ResourceType {
	case roleResourceType.Id:
		err := o.client.GrantRole(ctx, userID, role)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	default:
		return nil, nil, fmt.Errorf("baton-demo: unknown resource type")
	}
}

func (o *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	role := grant.Entitlement.Resource.Id.Resource
	userID := grant.Principal.Id.Resource

	switch grant.Entitlement.Resource.Id.ResourceType {
	case roleResourceType.Id:
		err := o.client.RevokeRole(ctx, userID, role)
		if err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown resource type")
	}
}

func newRoleBuilder(client *client.Client) *roleBuilder {
	return &roleBuilder{
		client: client,
	}
}
