package connector

import (
	"context"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

// nhiType maps the client nhi-type string to the proto enum.
func nhiType(s string) v2.NonHumanIdentityTrait_NhiType {
	switch s {
	case client.NHITypeAppRegistration:
		return v2.NonHumanIdentityTrait_NHI_TYPE_APP_REGISTRATION
	case client.NHITypeAssumableRole:
		return v2.NonHumanIdentityTrait_NHI_TYPE_ASSUMABLE_ROLE
	case client.NHITypeManagedIdentity:
		return v2.NonHumanIdentityTrait_NHI_TYPE_MANAGED_IDENTITY
	default:
		return v2.NonHumanIdentityTrait_NHI_TYPE_UNSPECIFIED
	}
}

// nhiBuilder syncs non-human identities of a single kind ("app" → TRAIT_APP,
// "role" → TRAIT_ROLE). Both kinds carry a NonHumanIdentityTrait.
type nhiBuilder struct {
	client       *client.Client
	kind         string
	resourceType *v2.ResourceType
}

var _ connectorbuilder.ResourceSyncerV2 = (*nhiBuilder)(nil)

func (o *nhiBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

func nhiResource(n *client.NHI, rt *v2.ResourceType, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	opts := []resource.ResourceOption{
		resource.WithParentResourceID(parentResourceID),
		resource.WithNHIType(nhiType(n.NhiType), n.NhiDetail),
		resource.WithResourceProfile(map[string]any{
			"nhi_type": n.NhiType,
		}),
	}
	switch n.Kind {
	case client.NHIKindRole:
		opts = append(opts, resource.WithRoleTrait())
	default:
		opts = append(opts, resource.WithAppTrait())
	}
	return resource.NewResource(n.Name, rt, n.Id, opts...)
}

func (o *nhiBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	nhiList, nextPageToken, err := o.client.ListNHIs(ctx, o.kind, &ops.PageToken)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Resource
	for _, n := range nhiList {
		r, err := nhiResource(n, o.resourceType, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		ret = append(ret, r)
	}

	return ret, &resource.SyncOpResults{NextPageToken: nextPageToken}, nil
}

func (o *nhiBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	n, err := o.client.GetNHI(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	r, err := nhiResource(n, o.resourceType, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return r, nil, nil
}

func (o *nhiBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *nhiBuilder) Grants(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func newNHIAppBuilder(cl *client.Client) *nhiBuilder {
	return &nhiBuilder{client: cl, kind: client.NHIKindApp, resourceType: nhiAppResourceType}
}

func newAssumableRoleBuilder(cl *client.Client) *nhiBuilder {
	return &nhiBuilder{client: cl, kind: client.NHIKindRole, resourceType: assumableRoleResourceType}
}
