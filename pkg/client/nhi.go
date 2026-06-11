package client

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pageOffset reads an offset-based page token, returning the limit and offset
// to use for a query. It mirrors the convention used by ListUsers.
func pageOffset(pToken *pagination.Token, defaultLimit int) (int, int, error) {
	limit := defaultLimit
	offset := 0
	if pToken != nil {
		if pToken.Size > 0 {
			limit = pToken.Size
		}
		if pToken.Token != "" {
			var err error
			offset, err = strconv.Atoi(pToken.Token)
			if err != nil {
				return 0, 0, err
			}
		}
	}
	if offset < 0 {
		return 0, 0, status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	return limit, offset, nil
}

// ---- Secrets (TRAIT_SECRET / K1) ----

var secretSelectColumns = []interface{}{
	"id", "name", "credential_type", "credential_detail", "identity_id",
	"created_at", "expires_at", "last_used_at", "updated_at",
}

func (c *Client) rowToSecret(_ context.Context, row scannable) (*Secret, error) {
	secret := &Secret{}
	identityID := sql.NullString{}
	expiresAt := sql.NullTime{}
	lastUsedAt := sql.NullTime{}
	err := row.Scan(
		&secret.Id, &secret.Name, &secret.CredentialType, &secret.CredentialDetail,
		&identityID, &secret.CreatedAt, &expiresAt, &lastUsedAt, &secret.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	secret.IdentityID = identityID.String
	if expiresAt.Valid {
		t := expiresAt.Time
		secret.ExpiresAt = &t
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		secret.LastUsedAt = &t
	}
	return secret, nil
}

// ListSecrets returns secrets from the database, paginated by an offset token.
func (c *Client) ListSecrets(ctx context.Context, pToken *pagination.Token) ([]*Secret, string, error) {
	if err := c.validateDB(); err != nil {
		return nil, "", err
	}

	limit, offset, err := pageOffset(pToken, 500)
	if err != nil {
		return nil, "", err
	}

	q := c.db.From(secrets.Name()).Prepared(true).
		Select(secretSelectColumns...).
		Order(goqu.C("id").Asc()).
		Limit(uint(limit)).  //nolint:gosec // limit validated > 0
		Offset(uint(offset)) //nolint:gosec // offset validated >= 0

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	out := []*Secret{}
	for rows.Next() {
		s, err := c.rowToSecret(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, s)
	}

	nextPageToken := ""
	if len(out) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}
	return out, nextPageToken, nil
}

// GetSecret returns a single secret by id.
func (c *Client) GetSecret(ctx context.Context, secretID string) (*Secret, error) {
	if err := c.validateDB(); err != nil {
		return nil, err
	}

	q := c.db.From(secrets.Name()).Prepared(true).
		Select(secretSelectColumns...).
		Where(goqu.C("id").Eq(secretID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToSecret(ctx, row)
}

// CreateSecret inserts a new secret. Used by the generator and by credential
// issuance/vending.
func (c *Client) CreateSecret(ctx context.Context, s *Secret) (*Secret, error) {
	if err := c.validateDB(); err != nil {
		return nil, err
	}

	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = s.CreatedAt
	}

	row := goqu.Record{
		"id":                s.Id,
		"name":              s.Name,
		"credential_type":   s.CredentialType,
		"credential_detail": s.CredentialDetail,
		"identity_id":       nullString(s.IdentityID),
		"created_at":        s.CreatedAt,
		"expires_at":        nullTime(s.ExpiresAt),
		"last_used_at":      nullTime(s.LastUsedAt),
		"updated_at":        s.UpdatedAt,
	}
	q := c.db.Insert(secrets.Name()).Prepared(true).Rows(row).OnConflict(goqu.DoUpdate("id", row))
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	if _, err := c.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	return s, nil
}

// ---- Non-human identities (TRAIT_APP / TRAIT_ROLE + NonHumanIdentityTrait / K3) ----

var nhiSelectColumns = []interface{}{
	"id", "name", "kind", "nhi_type", "nhi_detail", "created_at", "updated_at",
}

func (c *Client) rowToNHI(_ context.Context, row scannable) (*NHI, error) {
	nhi := &NHI{}
	detail := sql.NullString{}
	err := row.Scan(&nhi.Id, &nhi.Name, &nhi.Kind, &nhi.NhiType, &detail, &nhi.CreatedAt, &nhi.UpdatedAt)
	if err != nil {
		return nil, err
	}
	nhi.NhiDetail = detail.String
	return nhi, nil
}

// ListNHIs returns non-human identities of the given kind ("app" or "role"),
// paginated by an offset token.
func (c *Client) ListNHIs(ctx context.Context, kind string, pToken *pagination.Token) ([]*NHI, string, error) {
	if err := c.validateDB(); err != nil {
		return nil, "", err
	}

	limit, offset, err := pageOffset(pToken, 500)
	if err != nil {
		return nil, "", err
	}

	q := c.db.From(nhis.Name()).Prepared(true).
		Select(nhiSelectColumns...).
		Where(goqu.C("kind").Eq(kind)).
		Order(goqu.C("id").Asc()).
		Limit(uint(limit)).  //nolint:gosec // limit validated > 0
		Offset(uint(offset)) //nolint:gosec // offset validated >= 0

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	out := []*NHI{}
	for rows.Next() {
		n, err := c.rowToNHI(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, n)
	}

	nextPageToken := ""
	if len(out) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}
	return out, nextPageToken, nil
}

// GetNHI returns a single non-human identity by id.
func (c *Client) GetNHI(ctx context.Context, nhiID string) (*NHI, error) {
	if err := c.validateDB(); err != nil {
		return nil, err
	}

	q := c.db.From(nhis.Name()).Prepared(true).
		Select(nhiSelectColumns...).
		Where(goqu.C("id").Eq(nhiID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToNHI(ctx, row)
}

// ---- Agents (TRAIT_AGENT) ----

var agentSelectColumns = []interface{}{
	"id", "name", "status", "identity_id", "profile", "created_at", "updated_at",
}

func (c *Client) rowToAgent(_ context.Context, row scannable) (*Agent, error) {
	agent := &Agent{}
	identityID := sql.NullString{}
	profileBytes := []byte{}
	err := row.Scan(&agent.Id, &agent.Name, &agent.Status, &identityID, &profileBytes, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, err
	}
	agent.IdentityID = identityID.String
	if len(profileBytes) > 0 {
		if err := json.Unmarshal(profileBytes, &agent.Profile); err != nil {
			return nil, err
		}
	} else {
		agent.Profile = make(map[string]string)
	}
	return agent, nil
}

// ListAgents returns agents from the database, paginated by an offset token.
func (c *Client) ListAgents(ctx context.Context, pToken *pagination.Token) ([]*Agent, string, error) {
	if err := c.validateDB(); err != nil {
		return nil, "", err
	}

	limit, offset, err := pageOffset(pToken, 500)
	if err != nil {
		return nil, "", err
	}

	q := c.db.From(agents.Name()).Prepared(true).
		Select(agentSelectColumns...).
		Order(goqu.C("id").Asc()).
		Limit(uint(limit)).  //nolint:gosec // limit validated > 0
		Offset(uint(offset)) //nolint:gosec // offset validated >= 0

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	out := []*Agent{}
	for rows.Next() {
		a, err := c.rowToAgent(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, a)
	}

	nextPageToken := ""
	if len(out) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}
	return out, nextPageToken, nil
}

// GetAgent returns a single agent by id.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	if err := c.validateDB(); err != nil {
		return nil, err
	}

	q := c.db.From(agents.Name()).Prepared(true).
		Select(agentSelectColumns...).
		Where(goqu.C("id").Eq(agentID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToAgent(ctx, row)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}
