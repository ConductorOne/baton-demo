package connector

import (
	"context"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	entitlementSdk "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	demoAppResourceID  = "demo-app"
	demoAppDisplayName = "Demo App"

	// appAccessEntitlement is the slug of the "access" entitlement on the demo
	// app resource. loginUsageEventFeed's UsageEvents target this resource, and
	// C1's usage uplift reads those usage principals through this entitlement
	// (same pattern as baton-dropbox / baton-okta / baton-aws).
	appAccessEntitlement = "access"
)

// appBuilder syncs a single static "Demo App" App resource that the last-login
// usage-event feed attaches its events to. It mirrors baton-dropbox's appBuilder:
// the resource exists only to give loginUsageEventFeed a synced TargetResource.
type appBuilder struct{}

var _ connectorbuilder.ResourceSyncerV2 = (*appBuilder)(nil)

func newAppBuilder() *appBuilder {
	return &appBuilder{}
}

func (b *appBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return demoAppResourceType
}

func (b *appBuilder) List(_ context.Context, _ *v2.ResourceId, _ resourceSdk.SyncOpAttrs) ([]*v2.Resource, *resourceSdk.SyncOpResults, error) {
	res, err := resourceSdk.NewAppResource(demoAppDisplayName, demoAppResourceType, demoAppResourceID, nil)
	if err != nil {
		return nil, nil, err
	}
	return []*v2.Resource{res}, &resourceSdk.SyncOpResults{}, nil
}

func (b *appBuilder) Get(_ context.Context, _ *v2.ResourceId, _ *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	res, err := resourceSdk.NewAppResource(demoAppDisplayName, demoAppResourceType, demoAppResourceID, nil)
	if err != nil {
		return nil, nil, err
	}
	return res, nil, nil
}

// StaticEntitlements returns a single "access" assignment entitlement on the
// (singleton) demo app resource. C1's usage uplift iterates an app's App-trait
// entitlements and reads usage principals keyed to each entitlement's resource,
// so loginUsageEventFeed's UsageEvents only surface if this entitlement exists.
func (b *appBuilder) StaticEntitlements(_ context.Context, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	appResource, err := resourceSdk.NewAppResource(demoAppDisplayName, demoAppResourceType, demoAppResourceID, nil)
	if err != nil {
		return nil, nil, err
	}
	return []*v2.Entitlement{
		entitlementSdk.NewAssignmentEntitlement(
			appResource,
			appAccessEntitlement,
			entitlementSdk.WithGrantableTo(userResourceType),
			entitlementSdk.WithDisplayName("Demo App Access"),
			entitlementSdk.WithDescription("Has access to the Demo App"),
		),
	}, nil, nil
}

func (b *appBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	return nil, nil, nil
}

func (b *appBuilder) Grants(_ context.Context, _ *v2.Resource, _ resourceSdk.SyncOpAttrs) ([]*v2.Grant, *resourceSdk.SyncOpResults, error) {
	return nil, nil, nil
}
