package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userEventFeed struct {
	client *client.Client
}

var _ connectorbuilder.EventFeed = (*userEventFeed)(nil)

func (d *userEventFeed) EventFeedMetadata(ctx context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  "user-change-feed",
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_RESOURCE_CHANGE},
	}
}

func (d *userEventFeed) ListEvents(
	ctx context.Context,
	earliestEvent *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	events := []*v2.Event{}

	var occurredAt time.Time
	if earliestEvent != nil {
		occurredAt = earliestEvent.AsTime()
	}
	users, nextPageToken, err := d.client.ListUsersByUpdatedAt(ctx, occurredAt, sToken)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, user := range users {
		userResource, err := userResource(user, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("user-change-%s-%s", user.UpdatedAt.Format(time.RFC3339Nano), user.Id),
			OccurredAt: timestamppb.New(user.UpdatedAt),
			Event: &v2.Event_ResourceChangeEvent{
				ResourceChangeEvent: &v2.ResourceChangeEvent{
					ResourceId:       userResource.GetId(),
					ParentResourceId: userResource.GetParentResourceId(),
				},
			},
		})
	}

	streamState := &pagination.StreamState{
		Cursor:  nextPageToken,
		HasMore: nextPageToken != "",
	}

	return events, streamState, nil, nil
}

func newUserEventFeed(client *client.Client) *userEventFeed {
	return &userEventFeed{
		client: client,
	}
}
