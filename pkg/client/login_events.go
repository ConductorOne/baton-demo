package client

import (
	"context"
	"sort"
	"time"

	"github.com/conductorone/baton-sdk/pkg/pagination"
)

// LoginEvent is a single synthetic "successful login" for a human user. It is
// the demo analog of a Dropbox team_log login_success event: it carries just
// enough to build a UsageEvent (actor identity + email + when).
//
// Rank is the 1-based position of this login in the user's login history
// (1 == oldest, LoginEventsPerUser == newest). The feed groups events by Rank
// into "rounds" and emits one round per page, so Rank directly controls which
// login lands in which feed batch — that is the lever the ordering scenarios
// pull on.
type LoginEvent struct {
	UserID      string
	Email       string
	DisplayName string
	OccurredAt  time.Time
	Rank        int
}

// seedLoginEvents synthesizes LoginEventsPerUser logins for every human user,
// at strictly increasing timestamps spaced LoginEventSpacingSeconds apart, with
// the newest anchored just before `now` (so every event falls inside the feed's
// 24h catch-up window and none are in the future).
//
// Because every user gets the same number of logins at the same ranks, "round
// r" (all users' r-th login) is a clean cross-section: emitting rounds oldest
// -> newest across separate pages puts each user's newer logins in later
// batches, which is exactly what triggers C1's throttle-ordering bug.
func (c *Client) seedLoginEvents(ctx context.Context, now time.Time) error {
	perUser := c.config.LoginEventsPerUser
	if perUser <= 0 {
		perUser = 3
	}
	spacing := time.Duration(c.config.LoginEventSpacingSeconds) * time.Second
	if spacing <= 0 {
		spacing = time.Minute
	}

	users, err := c.allHumanUsers(ctx)
	if err != nil {
		return err
	}

	// Anchor the newest login at now-1m; walk backwards by `spacing` per rank so
	// rank 1 is the oldest and rank==perUser is the newest.
	newest := now.Add(-time.Minute)
	events := make([]LoginEvent, 0, len(users)*perUser)
	for _, u := range users {
		for rank := 1; rank <= perUser; rank++ {
			offset := time.Duration(perUser-rank) * spacing
			events = append(events, LoginEvent{
				UserID:      u.Id,
				Email:       u.Email,
				DisplayName: u.Name,
				OccurredAt:  newest.Add(-offset),
				Rank:        rank,
			})
		}
	}

	// Stable order: oldest first, then by user, so callers see a deterministic
	// stream. Emission order is applied later, in the feed.
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].OccurredAt.Equal(events[j].OccurredAt) {
			return events[i].OccurredAt.Before(events[j].OccurredAt)
		}
		return events[i].UserID < events[j].UserID
	})

	c.loginEvents = events
	return nil
}

// allHumanUsers pages through ListUsers and returns every human-account user.
func (c *Client) allHumanUsers(ctx context.Context) ([]*User, error) {
	var out []*User
	pToken := &pagination.Token{Size: 500}
	for {
		users, next, err := c.ListUsers(ctx, pToken)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			if u.AccountType == AccountTypeHuman && u.Email != "" {
				out = append(out, u)
			}
		}
		if next == "" {
			break
		}
		pToken = &pagination.Token{Size: 500, Token: next}
	}
	return out, nil
}

// LoginEventRounds returns the number of rounds (== max Rank == logins per
// user) in the seeded data. Zero when no logins were seeded.
func (c *Client) LoginEventRounds() int {
	maxRank := 0
	for _, e := range c.loginEvents {
		if e.Rank > maxRank {
			maxRank = e.Rank
		}
	}
	return maxRank
}

// LoginEventsForRound returns every user's login at the given rank whose
// timestamp is strictly after `after` (the feed watermark). Ordering within the
// returned slice is by user id; the feed does not depend on it.
func (c *Client) LoginEventsForRound(round int, after time.Time) []LoginEvent {
	var out []LoginEvent
	for _, e := range c.loginEvents {
		if e.Rank == round && e.OccurredAt.After(after) {
			out = append(out, e)
		}
	}
	return out
}

// MaxLoginEventTime returns the newest login timestamp strictly after `after`,
// or zero if there are none. Used to advance the feed watermark.
func (c *Client) MaxLoginEventTime(after time.Time) time.Time {
	var maxTS time.Time
	for _, e := range c.loginEvents {
		if e.OccurredAt.After(after) && e.OccurredAt.After(maxTS) {
			maxTS = e.OccurredAt
		}
	}
	return maxTS
}
