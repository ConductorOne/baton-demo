package connector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	loginEventFeedID = "demo_login_event_feed"

	// EmissionOldestFirst emits login pages in ascending time order (each user's
	// oldest login first, newest last). Across separate feed batches this is the
	// pattern that reproduces C1's throttle-ordering bug: the first batch writes
	// a user's oldest login and stamps UpdatedAt=now, so a pre-fix consumer
	// throttles away every newer login that follows within the 4h window.
	EmissionOldestFirst = "oldest-first"
	// EmissionReverseCursor emits newest first, mirroring baton-dropbox's
	// reverse-cursor mitigation: the newest login lands in the first batch, so
	// even a pre-fix consumer records the correct value and throttles the rest.
	EmissionReverseCursor = "reverse-cursor"

	// defaultCatchUpWindow bounds how far back the first drain (or one recovering
	// from an empty watermark) reaches. All synthetic logins are seeded inside
	// this window.
	defaultCatchUpWindow = 24 * time.Hour
)

// loginUsageEventFeed emits EVENT_TYPE_USAGE events for successful demo logins,
// each targeting the static demo app resource with the logging-in user as the
// actor. It is the demo analog of baton-dropbox's loginEventFeed and exists to
// drive C1's baton-feed-consumer (HandleUsageEvents / getUsagePrincipal) with
// fully controllable ordering and timing.
//
// Emission is page-per-round: "round r" is every user's r-th login. The feed
// emits one round per ListEvents call, so each round becomes a separate
// HandleUsageEvents batch on the C1 side. The emission-order config decides
// whether rounds go oldest->newest (bug-reproducing) or newest->oldest
// (mitigated); the fix under test must yield the newest login either way.
type loginUsageEventFeed struct {
	client        *client.Client
	emissionOrder string
}

var _ connectorbuilder.EventFeed = (*loginUsageEventFeed)(nil)

func newLoginUsageEventFeed(cl *client.Client, emissionOrder string) *loginUsageEventFeed {
	if emissionOrder != EmissionReverseCursor {
		emissionOrder = EmissionOldestFirst
	}
	return &loginUsageEventFeed{client: cl, emissionOrder: emissionOrder}
}

func (f *loginUsageEventFeed) EventFeedMetadata(_ context.Context) *v2.EventFeedMetadata {
	return &v2.EventFeedMetadata{
		Id:                  loginEventFeedID,
		SupportedEventTypes: []v2.EventType{v2.EventType_EVENT_TYPE_USAGE},
	}
}

// loginFeedToken is the state threaded between ListEvents calls for one drain.
type loginFeedToken struct {
	// Watermark is the RFC3339Nano timestamp the current drain started from;
	// only logins strictly after it are in scope. Promoted to the newest event
	// seen once the drain finishes.
	Watermark string `json:"watermark,omitempty"`
	// PendingRounds is the ordered list of round ranks still to emit, in the
	// configured emission order. Computed once at drain start.
	PendingRounds []int `json:"pending_rounds,omitempty"`
	// NewestSeen is the newest event timestamp observed so far this drain; it
	// becomes the next drain's watermark.
	NewestSeen string `json:"newest_seen,omitempty"`
	// Started is true once the drain has been planned (PendingRounds computed).
	Started bool `json:"started,omitempty"`
}

func (t *loginFeedToken) marshal() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func unmarshalLoginFeedToken(sToken *pagination.StreamToken) (*loginFeedToken, error) {
	t := &loginFeedToken{}
	if sToken != nil && sToken.Cursor != "" {
		data, err := base64.StdEncoding.DecodeString(sToken.Cursor)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, t); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (f *loginUsageEventFeed) ListEvents(
	ctx context.Context,
	startAt *timestamppb.Timestamp,
	sToken *pagination.StreamToken,
) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	token, err := unmarshalLoginFeedToken(sToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-demo: failed to unmarshal login feed token: %w", err)
	}

	// Plan the drain on the first call: resolve the watermark and the ordered
	// list of rounds to emit. Subsequent calls just pop the next round, so a
	// mid-drain change to startAt cannot disturb the plan.
	if !token.Started {
		// The watermark is the latest of the caller's start-at and any watermark
		// carried on a prior terminal token, floored at the 24h catch-up window.
		// Taking the max means a completed drain never re-emits, whether C1
		// advances start-at (via the checkpoint) or just threads our cursor.
		watermark := time.Now().Add(-defaultCatchUpWindow)
		if startAt != nil && startAt.IsValid() && startAt.AsTime().After(watermark) {
			watermark = startAt.AsTime()
		}
		if carried, err := time.Parse(time.RFC3339Nano, token.Watermark); err == nil && carried.After(watermark) {
			watermark = carried
		}
		token.Watermark = watermark.UTC().Format(time.RFC3339Nano)
		token.NewestSeen = token.Watermark
		token.PendingRounds = f.plannedRounds(watermark)
		token.Started = true
	}

	watermark, err := time.Parse(time.RFC3339Nano, token.Watermark)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-demo: failed to parse login feed watermark: %w", err)
	}

	// Nothing left to emit -> finish the drain, promoting the watermark past the
	// newest event so the next drain is empty.
	if len(token.PendingRounds) == 0 {
		return f.finishDrain(l, token)
	}

	round := token.PendingRounds[0]
	token.PendingRounds = token.PendingRounds[1:]

	logins := f.client.LoginEventsForRound(round, watermark)
	events, newest := f.buildUsageEvents(logins)
	if newest.Format(time.RFC3339Nano) > token.NewestSeen {
		token.NewestSeen = newest.Format(time.RFC3339Nano)
	}

	l.Info("baton-demo: emitted login usage-event round",
		zap.Int("round", round),
		zap.String("emission_order", f.emissionOrder),
		zap.Int("events", len(events)),
		zap.Int("rounds_remaining", len(token.PendingRounds)),
	)

	if len(token.PendingRounds) == 0 {
		return f.finishDrainWith(l, token, events)
	}

	cursor, err := token.marshal()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-demo: failed to marshal login feed token: %w", err)
	}
	return events, &pagination.StreamState{Cursor: cursor, HasMore: true}, nil, nil
}

// plannedRounds returns the round ranks in the configured emission order,
// restricted to rounds that actually have logins after the watermark.
func (f *loginUsageEventFeed) plannedRounds(watermark time.Time) []int {
	rounds := f.client.LoginEventRounds()
	var out []int
	for r := 1; r <= rounds; r++ {
		if len(f.client.LoginEventsForRound(r, watermark)) > 0 {
			out = append(out, r)
		}
	}
	if f.emissionOrder == EmissionReverseCursor {
		sort.Sort(sort.Reverse(sort.IntSlice(out)))
	}
	return out
}

// buildUsageEvents converts a round of logins into UsageEvents targeting the
// demo app resource, and returns the newest OccurredAt in the round.
func (f *loginUsageEventFeed) buildUsageEvents(logins []client.LoginEvent) ([]*v2.Event, time.Time) {
	events := make([]*v2.Event, 0, len(logins))
	var newest time.Time
	for _, login := range logins {
		if login.OccurredAt.After(newest) {
			newest = login.OccurredAt
		}
		userTrait, err := resourceSdk.NewUserTrait(resourceSdk.WithEmail(login.Email, true))
		if err != nil {
			// A malformed email shouldn't drop the whole round; skip this login.
			continue
		}
		events = append(events, &v2.Event{
			Id:         fmt.Sprintf("%s-%s", login.UserID, login.OccurredAt.Format(time.RFC3339Nano)),
			OccurredAt: timestamppb.New(login.OccurredAt),
			Event: &v2.Event_UsageEvent{
				UsageEvent: &v2.UsageEvent{
					TargetResource: &v2.Resource{
						Id: &v2.ResourceId{
							ResourceType: demoAppResourceType.Id,
							Resource:     demoAppResourceID,
						},
						DisplayName: demoAppDisplayName,
					},
					ActorResource: &v2.Resource{
						Id: &v2.ResourceId{
							ResourceType: userResourceType.Id,
							Resource:     login.UserID,
						},
						DisplayName: login.DisplayName,
						Annotations: annotations.New(userTrait),
					},
				},
			},
		})
	}
	return events, newest
}

// finishDrain closes out a drain that has no events to emit on this call.
func (f *loginUsageEventFeed) finishDrain(l *zap.Logger, token *loginFeedToken) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	return f.finishDrainWith(l, token, nil)
}

// finishDrainWith returns a terminal (HasMore=false) state carrying `events`,
// advancing the watermark to the newest event seen this drain so the next drain
// starts just past it.
func (f *loginUsageEventFeed) finishDrainWith(l *zap.Logger, token *loginFeedToken, events []*v2.Event) ([]*v2.Event, *pagination.StreamState, annotations.Annotations, error) {
	next := &loginFeedToken{Watermark: token.NewestSeen}
	cursor, err := next.marshal()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-demo: failed to marshal terminal login feed token: %w", err)
	}
	l.Info("baton-demo: login usage-event drain complete",
		zap.String("emission_order", f.emissionOrder),
		zap.String("watermark", token.NewestSeen),
	)
	return events, &pagination.StreamState{Cursor: cursor, HasMore: false}, nil, nil
}
