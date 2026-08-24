package connector

import (
	"context"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type agentBuilder struct {
	client *client.Client
}

var _ connectorbuilder.ResourceSyncerV2 = (*agentBuilder)(nil)

func (o *agentBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return agentResourceType
}

// agentStatus maps the client agent-status string to the proto enum.
func agentStatus(s string) v2.Status_ResourceStatus {
	switch s {
	case client.AgentStatusReady:
		return v2.Status_RESOURCE_STATUS_ENABLED
	case client.AgentStatusDisabled:
		return v2.Status_RESOURCE_STATUS_DISABLED
	case client.AgentStatusDeleted:
		return v2.Status_RESOURCE_STATUS_DELETED
	default:
		return v2.Status_RESOURCE_STATUS_UNSPECIFIED
	}
}

func agentResource(a *client.Agent, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := make(map[string]any, len(a.Profile))
	for k, v := range a.Profile {
		profile[k] = v
	}

	agentOpts := []resource.AgentTraitOption{}
	// The identity the agent authenticates as (a service-account user).
	if a.IdentityID != "" {
		identityID, err := resource.NewResourceID(userResourceType, a.IdentityID)
		if err != nil {
			return nil, err
		}
		agentOpts = append(agentOpts, resource.WithAgentIdentityResourceID(identityID))
	}

	return resource.NewResource(
		a.Name,
		agentResourceType,
		a.Id,
		resource.WithAgentTrait(agentOpts...),
		resource.WithParentResourceID(parentResourceID),
		resource.WithResourceStatus(agentStatus(a.Status), a.Status),
		resource.WithResourceProfile(profile),
		resource.WithResourceCreatedAt(a.CreatedAt),
	)
}

func (o *agentBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	agentList, nextPageToken, err := o.client.ListAgents(ctx, &ops.PageToken)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Resource
	for _, a := range agentList {
		r, err := agentResource(a, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		ret = append(ret, r)
	}

	return ret, &resource.SyncOpResults{NextPageToken: nextPageToken}, nil
}

func (o *agentBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	a, err := o.client.GetAgent(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	r, err := agentResource(a, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return r, nil, nil
}

func (o *agentBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *agentBuilder) Grants(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func newAgentBuilder(client *client.Client) *agentBuilder {
	return &agentBuilder{client: client}
}
