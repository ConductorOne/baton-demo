package connector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conductorone/baton-demo/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// loginFeedConfig returns a config with the last-login feed enabled and a small,
// deterministic estate: a fixed number of human users, each with a fixed number
// of logins spaced a fixed interval apart.
func loginFeedConfig(t *testing.T, emission string, users, perUser, spacingSeconds int) *config.Demo {
	t.Helper()
	dc := estateConfig(t)
	dc.Users = users
	dc.SyncUserLastLogin = true
	dc.LoginEmissionOrder = emission
	dc.LoginEventsPerUser = perUser
	dc.LoginEventSpacingSeconds = spacingSeconds
	return dc
}

// batch is one emitted feed page (one HandleUsageEvents batch on the C1 side).
type batch struct {
	events []*v2.Event
}

// driveFeed runs the feed exactly the way C1's baton-feed-consumer would: it
// calls ListEvents in a loop, threading the returned cursor, until HasMore is
// false. It returns one batch per page, in emission order.
func driveFeed(t *testing.T, feed *loginUsageEventFeed, startAt *timestamppb.Timestamp) []batch {
	t.Helper()
	ctx := context.Background()
	var batches []batch
	var cursor string
	for i := 0; i < 100; i++ { // safety bound
		events, state, _, err := feed.ListEvents(ctx, startAt, &pagination.StreamToken{Cursor: cursor})
		require.NoError(t, err)
		require.NotNil(t, state)
		batches = append(batches, batch{events: events})
		if !state.HasMore {
			return batches
		}
		cursor = state.Cursor
	}
	t.Fatal("feed did not terminate within safety bound")
	return nil
}

// newestPerUser returns the newest OccurredAt seen per actor user id across all
// batches (i.e. what the final LastLogin should be if the consumer never drops
// a newer value).
func newestPerUser(batches []batch) map[string]time.Time {
	out := map[string]time.Time{}
	for _, b := range batches {
		for _, e := range b.events {
			uid := e.GetUsageEvent().GetActorResource().GetId().GetResource()
			ts := e.OccurredAt.AsTime()
			if cur, ok := out[uid]; !ok || ts.After(cur) {
				out[uid] = ts
			}
		}
	}
	return out
}

func TestLoginFeed_OldestFirstOrdering(t *testing.T) {
	feed := newLoginUsageEventFeed(testClient(t, loginFeedConfig(t, EmissionOldestFirst, 3, 4, 60)), EmissionOldestFirst)
	batches := driveFeed(t, feed, nil)

	// One batch per round (login rank); 4 logins/user => 4 batches.
	require.Len(t, batches, 4)

	// Each batch's newest event must be strictly older than the next batch's:
	// oldest-first means batch i carries every user's i-th (older) login.
	var prev time.Time
	for i, b := range batches {
		require.NotEmpty(t, b.events, "batch %d empty", i)
		newest := batchNewest(b)
		if i > 0 {
			assert.True(t, prev.Before(newest), "batch %d should be newer than batch %d", i, i-1)
		}
		prev = newest
	}
}

func TestLoginFeed_ReverseCursorOrdering(t *testing.T) {
	feed := newLoginUsageEventFeed(testClient(t, loginFeedConfig(t, EmissionReverseCursor, 3, 4, 60)), EmissionReverseCursor)
	batches := driveFeed(t, feed, nil)
	require.Len(t, batches, 4)

	// reverse-cursor means the newest login lands in the FIRST batch; each
	// subsequent batch is strictly older.
	var prev time.Time
	for i, b := range batches {
		require.NotEmpty(t, b.events, "batch %d empty", i)
		newest := batchNewest(b)
		if i > 0 {
			assert.True(t, newest.Before(prev), "batch %d should be older than batch %d", i, i-1)
		}
		prev = newest
	}
}

func TestLoginFeed_EventShape(t *testing.T) {
	feed := newLoginUsageEventFeed(testClient(t, loginFeedConfig(t, EmissionOldestFirst, 2, 2, 60)), EmissionOldestFirst)
	batches := driveFeed(t, feed, nil)

	seen := 0
	for _, b := range batches {
		for _, e := range b.events {
			ue := e.GetUsageEvent()
			require.NotNil(t, ue)
			// Target is the static demo app.
			assert.Equal(t, demoAppResourceType.Id, ue.GetTargetResource().GetId().GetResourceType())
			assert.Equal(t, demoAppResourceID, ue.GetTargetResource().GetId().GetResource())
			// Actor is a user carrying a primary-email user trait.
			actor := ue.GetActorResource()
			assert.Equal(t, userResourceType.Id, actor.GetId().GetResourceType())
			assert.NotEmpty(t, actor.GetId().GetResource())
			ut := &v2.UserTrait{}
			annos := annotations.Annotations(actor.GetAnnotations())
			ok, err := annos.Pick(ut)
			require.NoError(t, err)
			require.True(t, ok, "actor must carry a UserTrait annotation")
			require.NotEmpty(t, ut.GetEmails())
			assert.True(t, ut.GetEmails()[0].GetIsPrimary())
			// OccurredAt set and Id present.
			assert.True(t, e.OccurredAt.IsValid())
			assert.NotEmpty(t, e.GetId())
			seen++
		}
	}
	assert.Equal(t, 2*2, seen, "expected users*perUser total events")
}

// TestLoginFeed_DrainTerminatesAndIsIdempotent verifies a full drain emits every
// login exactly once and that a follow-up drain starting from the advanced
// watermark emits nothing (no reprocessing).
func TestLoginFeed_DrainTerminatesAndIsIdempotent(t *testing.T) {
	feed := newLoginUsageEventFeed(testClient(t, loginFeedConfig(t, EmissionOldestFirst, 3, 3, 60)), EmissionOldestFirst)
	batches := driveFeed(t, feed, nil)

	total := 0
	var newest time.Time
	for _, b := range batches {
		total += len(b.events)
		if n := batchNewest(b); n.After(newest) {
			newest = n
		}
	}
	assert.Equal(t, 3*3, total)

	// Second drain from the newest watermark -> nothing new.
	second := driveFeed(t, feed, timestamppb.New(newest))
	got := 0
	for _, b := range second {
		got += len(b.events)
	}
	assert.Zero(t, got, "a drain from the advanced watermark must emit no events")
}

// TestLoginFeed_NewestWinsRegardlessOfOrder is the crux: it replays the emitted
// batches through a faithful model of C1's getUsagePrincipal throttle and shows
// the fixed consumer always ends on the newest login, in BOTH emission orders —
// while the pre-fix consumer only gets it right under reverse-cursor.
func TestLoginFeed_NewestWinsRegardlessOfOrder(t *testing.T) {
	for _, tc := range []struct {
		emission          string
		preFixMatchesTrue bool // does the buggy consumer land on the newest?
	}{
		{EmissionOldestFirst, false},   // bug shows: oldest locks in, newer throttled
		{EmissionReverseCursor, true},  // mitigated: newest lands first
	} {
		t.Run(tc.emission, func(t *testing.T) {
			feed := newLoginUsageEventFeed(testClient(t, loginFeedConfig(t, tc.emission, 3, 4, 60)), tc.emission)
			batches := driveFeed(t, feed, nil)
			want := newestPerUser(batches)

			fixed := replayConsumer(batches, true /*advancingWins*/)
			preFix := replayConsumer(batches, false)

			assert.Equal(t, want, fixed, "fixed consumer must always land on the newest login")
			if tc.preFixMatchesTrue {
				assert.Equal(t, want, preFix, "under reverse-cursor even the pre-fix consumer is correct")
			} else {
				assert.NotEqual(t, want, preFix, "under oldest-first the pre-fix consumer must be stale (bug)")
			}
		})
	}
}

func batchNewest(b batch) time.Time {
	var newest time.Time
	for _, e := range b.events {
		if ts := e.OccurredAt.AsTime(); ts.After(newest) {
			newest = ts
		}
	}
	return newest
}

// replayConsumer models C1's getUsagePrincipal across batches. Each batch is a
// separate HandleUsageEvents call; within a batch the max login per principal is
// kept (order-independent), then a stateful store is updated. The store stamps
// UpdatedAt=now on every write, so a second batch arriving "immediately" is
// always inside the throttle window.
//
//   - advancingWins=true  models the fix: a strictly newer login always writes.
//   - advancingWins=false models the bug: any write within the window is
//     throttled BEFORE the freshness check, so newer logins are dropped.
func replayConsumer(batches []batch, advancingWins bool) map[string]time.Time {
	type row struct {
		lastLogin   time.Time
		everWritten bool
	}
	store := map[string]*row{}
	for _, b := range batches {
		// In-batch dedup: keep the newest login per principal.
		batchMax := map[string]time.Time{}
		for _, e := range b.events {
			uid := e.GetUsageEvent().GetActorResource().GetId().GetResource()
			ts := e.OccurredAt.AsTime()
			if cur, ok := batchMax[uid]; !ok || ts.After(cur) {
				batchMax[uid] = ts
			}
		}
		for uid, ts := range batchMax {
			r := store[uid]
			if r == nil {
				r = &row{}
				store[uid] = r
			}
			advances := !r.everWritten || ts.After(r.lastLogin)
			if r.everWritten && !advancingWins {
				// Pre-fix: recently-written rows are throttled before the
				// freshness check, so nothing newer ever lands.
				continue
			}
			if advances {
				r.lastLogin = ts
				r.everWritten = true
			}
		}
	}
	out := map[string]time.Time{}
	for uid, r := range store {
		out[uid] = r.lastLogin
	}
	return out
}
