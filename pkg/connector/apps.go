package connector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var appAccessEntitlement = "access"

type appBuilder struct {
	client *client.Client
}

func (o *appBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return appResourceType
}

func appResource(a *client.App, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := make(map[string]any)
	profile["app_name"] = a.Name
	profile["member_count"] = len(a.Members)
	profile["child_group_count"] = len(a.ChildGroups)
	profile["created_at"] = a.CreatedAt.Format(time.RFC3339)
	profile["updated_at"] = a.UpdatedAt.Format(time.RFC3339)

	return resource.NewAppResource(
		a.Name,
		appResourceType,
		a.Id,
		[]resource.AppTraitOption{resource.WithAppProfile(profile)},
		resource.WithParentResourceID(parentResourceID),
	)
}

func (o *appBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	apps, err := o.client.ListApps(ctx)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Resource
	for _, a := range apps {
		app, err := appResource(a, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		ret = append(ret, app)
	}

	return ret, nil, nil
}

func (o *appBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	app, err := o.client.GetApp(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	r, err := appResource(app, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return r, nil, nil
}

func (o *appBuilder) Entitlements(ctx context.Context, r *v2.Resource, ops resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	access := sdkEntitlement.NewAssignmentEntitlement(r, appAccessEntitlement, sdkEntitlement.WithGrantableTo(userResourceType))
	access.Description = fmt.Sprintf("Has access to the %s app", r.DisplayName)

	return []*v2.Entitlement{access}, nil, nil
}

func (o *appBuilder) Grants(ctx context.Context, r *v2.Resource, ops resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(ops.PageToken.Token)
	if err != nil {
		return nil, nil, err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: "members",
			Token:          "0",
		})
	}

	app, err := o.client.GetApp(ctx, r.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	limit := ops.PageToken.Size
	if limit == 0 {
		limit = 1000
	}

	ps := b.Current()

	offset, err := strconv.Atoi(ps.Token)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Grant

	switch ps.ResourceTypeID {
	case "members":
		end := min(offset+limit, len(app.Members))
		for _, memberID := range app.Members[offset:end] {
			pID, err := resource.NewResourceID(userResourceType, memberID)
			if err != nil {
				return nil, nil, err
			}
			ret = append(ret, sdkGrant.NewGrant(r, appAccessEntitlement, pID))
		}
		nextPage := ""
		if end < len(app.Members) {
			nextPage = strconv.Itoa(end)
		}
		nextPageToken, err := b.NextToken(nextPage)
		if err != nil {
			return nil, nil, err
		}
		return ret, &resource.SyncOpResults{NextPageToken: nextPageToken}, nil
	default:
		return nil, nil, fmt.Errorf("unknown resource type")
	}
}

func parseAppID(entitlementID string) (string, string, error) {
	parts := strings.Split(entitlementID, ":")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid app entitlement ID %s", entitlementID)
	}
	return parts[1], parts[2], nil
}

func (o *appBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	if principal.Id.ResourceType != userResourceType.Id {
		return nil, nil, fmt.Errorf("baton-demo: only users can have app access granted")
	}

	if entitlement.Resource.Id.ResourceType != appResourceType.Id {
		return nil, nil, fmt.Errorf("baton-demo: only apps can have access granted")
	}

	appID, _, err := parseAppID(entitlement.Id)
	if err != nil {
		return nil, nil, err
	}
	userID := principal.Id.Resource

	err = o.client.GrantAppAccess(ctx, appID, userID)
	if err != nil {
		return nil, nil, err
	}

	return nil, nil, nil
}

func (o *appBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	appID := grant.Entitlement.Resource.Id.Resource
	principalId := grant.Principal.Id

	if principalId.ResourceType != userResourceType.Id {
		return nil, status.Errorf(codes.InvalidArgument, "only users can have app access revoked")
	}

	err := o.client.RevokeAppAccess(ctx, appID, principalId.Resource)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func newAppBuilder(client *client.Client) *appBuilder {
	return &appBuilder{
		client: client,
	}
}
