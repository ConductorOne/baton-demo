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

// groupEventFeed

type groupEventFeed struct {
	client *client.Client
}

var _ connectorbuilder.EventFeed = (*groupEventFeed)(nil)

func (d *groupEventFeed) EventFeedMetadata(ctx context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  "group-change-feed",
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_RESOURCE_CHANGE},
	}
}

func (d *groupEventFeed) ListEvents(
	ctx context.Context,
	earliestEvent *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	events := []*v2.Event{}

	var occurredAt time.Time
	if earliestEvent != nil {
		occurredAt = earliestEvent.AsTime()
	}
	groups, nextPageToken, err := d.client.ListGroupsByUpdatedAt(ctx, occurredAt, sToken)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, group := range groups {
		groupRes, err := groupResource(group, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("group-change-%s-%s", group.UpdatedAt.Format(time.RFC3339Nano), group.Id),
			OccurredAt: timestamppb.New(group.UpdatedAt),
			Event: &v2.Event_ResourceChangeEvent{
				ResourceChangeEvent: &v2.ResourceChangeEvent{
					ResourceId:       groupRes.GetId(),
					ParentResourceId: groupRes.GetParentResourceId(),
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

func newGroupEventFeed(client *client.Client) *groupEventFeed {
	return &groupEventFeed{
		client: client,
	}
}

// roleEventFeed

type roleEventFeed struct {
	client *client.Client
}

var _ connectorbuilder.EventFeed = (*roleEventFeed)(nil)

func (d *roleEventFeed) EventFeedMetadata(ctx context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  "role-change-feed",
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_RESOURCE_CHANGE},
	}
}

func (d *roleEventFeed) ListEvents(
	ctx context.Context,
	earliestEvent *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	events := []*v2.Event{}

	var occurredAt time.Time
	if earliestEvent != nil {
		occurredAt = earliestEvent.AsTime()
	}
	roles, nextPageToken, err := d.client.ListRolesByUpdatedAt(ctx, occurredAt, sToken)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, role := range roles {
		roleRes, err := roleResource(role, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("role-change-%s-%s", role.UpdatedAt.Format(time.RFC3339Nano), role.Id),
			OccurredAt: timestamppb.New(role.UpdatedAt),
			Event: &v2.Event_ResourceChangeEvent{
				ResourceChangeEvent: &v2.ResourceChangeEvent{
					ResourceId:       roleRes.GetId(),
					ParentResourceId: roleRes.GetParentResourceId(),
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

func newRoleEventFeed(client *client.Client) *roleEventFeed {
	return &roleEventFeed{
		client: client,
	}
}

// projectEventFeed

type projectEventFeed struct {
	client *client.Client
}

var _ connectorbuilder.EventFeed = (*projectEventFeed)(nil)

func (d *projectEventFeed) EventFeedMetadata(ctx context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  "project-change-feed",
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_RESOURCE_CHANGE},
	}
}

func (d *projectEventFeed) ListEvents(
	ctx context.Context,
	earliestEvent *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	events := []*v2.Event{}

	var occurredAt time.Time
	if earliestEvent != nil {
		occurredAt = earliestEvent.AsTime()
	}
	projects, nextPageToken, err := d.client.ListProjectsByUpdatedAt(ctx, occurredAt, sToken)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, project := range projects {
		projectRes, err := projectResource(project, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("project-change-%s-%s", project.UpdatedAt.Format(time.RFC3339Nano), project.Id),
			OccurredAt: timestamppb.New(project.UpdatedAt),
			Event: &v2.Event_ResourceChangeEvent{
				ResourceChangeEvent: &v2.ResourceChangeEvent{
					ResourceId:       projectRes.GetId(),
					ParentResourceId: projectRes.GetParentResourceId(),
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

func newProjectEventFeed(client *client.Client) *projectEventFeed {
	return &projectEventFeed{
		client: client,
	}
}

// scopedRoleEventFeed

type scopedRoleEventFeed struct {
	client *client.Client
}

var _ connectorbuilder.EventFeed = (*scopedRoleEventFeed)(nil)

func (d *scopedRoleEventFeed) EventFeedMetadata(ctx context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  "scoped-role-change-feed",
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_RESOURCE_CHANGE},
	}
}

func (d *scopedRoleEventFeed) ListEvents(
	ctx context.Context,
	earliestEvent *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	events := []*v2.Event{}

	var occurredAt time.Time
	if earliestEvent != nil {
		occurredAt = earliestEvent.AsTime()
	}
	scopedRoles, nextPageToken, err := d.client.ListScopedRolesByUpdatedAt(ctx, occurredAt, sToken)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, sr := range scopedRoles {
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("scoped-role-change-%s-%s", sr.UpdatedAt.Format(time.RFC3339Nano), sr.Id),
			OccurredAt: timestamppb.New(sr.UpdatedAt),
			Event: &v2.Event_ResourceChangeEvent{
				ResourceChangeEvent: &v2.ResourceChangeEvent{
					ResourceId: &v2.ResourceId{
						ResourceType: scopedRoleResourceType.Id,
						Resource:     makeScopedRoleResourceID(sr.RoleId, sr.ProjectId),
					},
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

func newScopedRoleEventFeed(client *client.Client) *scopedRoleEventFeed {
	return &scopedRoleEventFeed{
		client: client,
	}
}
