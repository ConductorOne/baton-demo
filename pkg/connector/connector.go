package connector

import (
	"context"
	"io"

	"github.com/conductorone/baton-demo/pkg/client"
	"github.com/conductorone/baton-demo/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

type Demo struct {
	client *client.Client
}

var _ connectorbuilder.EventFeedsLimited = (*Demo)(nil)
var _ connectorbuilder.ConnectorBuilderV2 = (*Demo)(nil)

func (d *Demo) EventFeeds(ctx context.Context) []connectorbuilder.EventFeed {
	return []connectorbuilder.EventFeed{
		newUserEventFeed(d.client),
		newGroupEventFeed(d.client),
		newRoleEventFeed(d.client),
		newProjectEventFeed(d.client),
		newScopedRoleEventFeed(d.client),
	}
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Demo) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	return []connectorbuilder.ResourceSyncerV2{
		newUserBuilder(d.client),
		newGroupBuilder(d.client),
		newRoleBuilder(d.client),
		newScopedRoleBuilder(d.client),
		newProjectBuilder(d.client),
		newSecretBuilder(d.client),
		newNHIAppBuilder(d.client),
		newAssumableRoleBuilder(d.client),
		newAgentBuilder(d.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Demo) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Demo) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Demo",
		Description: "A demo connector",
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *Demo) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns a new instance of the Demo connector.
func New(ctx context.Context, dc *config.Demo) (*Demo, error) {
	cli, err := client.NewClient(ctx, dc)
	if err != nil {
		return nil, err
	}
	demo := &Demo{
		client: cli,
	}

	return demo, nil
}
