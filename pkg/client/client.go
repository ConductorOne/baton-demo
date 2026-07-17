package client

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/conductorone/baton-demo/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/doug-martin/goqu/v9"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// NOTE: required to register the dialect for goqu.
	//
	// If you remove this import, goqu.Dialect("sqlite3") will
	// return a copy of the default dialect, which is not what we want,
	// and allocates a ton of memory.
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
)

// Resource model
// Users are humans
// Groups can be assigned Users as Admins or Members
// Roles can be assigned directly to Users or to a Group
// ScopedRoles are roles that are scoped to a project
// Projects always have a single User as the owner, and can be assigned to Groups

const (
	minEvents = 50
	maxEvents = 100
)

// Account type values stored in the users.account_type column. They map to
// v2.UserTrait_AccountType in the connector layer.
const (
	AccountTypeHuman   = "human"
	AccountTypeService = "service"
	AccountTypeSystem  = "system"
)

type User struct {
	Id          string
	Name        string
	Email       string
	Enabled     bool
	AccountType string
	Attrs       map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Secret models a credential (TRAIT_SECRET / K1). IdentityID is the owning
// identity (a service-account user); when empty the secret is unowned.
type Secret struct {
	Id               string
	Name             string
	CredentialType   string
	CredentialDetail string
	IdentityID       string
	CreatedAt        time.Time
	ExpiresAt        *time.Time
	LastUsedAt       *time.Time
	UpdatedAt        time.Time
}

// NHI models a non-human identity (K3) carried on a TRAIT_APP or TRAIT_ROLE
// resource. Kind selects the resource type ("app" or "role"); NhiType is one
// of app_registration | assumable_role | managed_identity.
type NHI struct {
	Id        string
	Name      string
	Kind      string
	NhiType   string
	NhiDetail string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Agent models an autonomous non-human actor (TRAIT_AGENT) that authenticates
// as a service-account user (IdentityID).
type Agent struct {
	Id         string
	Name       string
	Status     string
	IdentityID string
	Profile    map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Secret credential-type values stored in the secrets.credential_type column.
// These are classification labels, not credential material.
const (
	CredentialTypeStaticSecret  = "static_secret"
	CredentialTypeAsymmetricKey = "asymmetric_key" //nolint:gosec // classification label, not a credential
	CredentialTypeCertificate   = "certificate"
)

// NHI kind values selecting the emitted resource type.
const (
	NHIKindApp  = "app"
	NHIKindRole = "role"
)

// NHI type values stored in the nhis.nhi_type column.
const (
	NHITypeAppRegistration = "app_registration"
	NHITypeAssumableRole   = "assumable_role"
	NHITypeManagedIdentity = "managed_identity"
)

// Agent status values stored in the agents.status column.
const (
	AgentStatusReady    = "ready"
	AgentStatusDisabled = "disabled"
	AgentStatusDeleted  = "deleted"
)

type Group struct {
	Id        string
	Name      string
	Admins    []string
	Members   []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Role struct {
	Id                string
	Name              string
	DirectAssignments []string
	GroupAssignments  []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ScopedRole struct {
	Id              string
	ProjectId       string
	RoleId          string
	UserAssignments []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Project struct {
	Id               string
	Name             string
	Owner            string
	GroupAssignments []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Password struct {
	Id       string
	Password string
	UserId   string
}

// Client is a simple example client. While this client would normally be responsible for communicating with an upstream.
// API, for this demo the client is only working with in-memory data.
type Client struct {
	db     *goqu.Database
	rawDB  *sql.DB
	config *config.Demo

	// loginEvents holds synthetic successful-login events, one set per human
	// user, generated at startup. They back the last-login usage-event feed
	// (see login_events.go) and are read-only after seeding.
	loginEvents []LoginEvent
}

func NewClient(ctx context.Context, dc *config.Demo) (*Client, error) {
	c := &Client{
		config: dc,
	}

	// Open the database file
	if c.config.DbFileName == "" {
		c.config.DbFileName = "baton-demo.db"
	}

	if _, err := os.Stat(c.config.DbFileName); os.IsNotExist(err) {
		c.config.InitDb = true
	}

	rawDB, err := sql.Open("sqlite", c.config.DbFileName)
	if err != nil {
		return nil, err
	}

	db := goqu.New("sqlite3", rawDB)
	c.db = db
	c.rawDB = rawDB

	// ensure all schemas exist
	for _, t := range allTableDescriptors {
		queries, args := t.Schema()
		for _, query := range queries {
			_, err := c.db.Exec(query, args...)
			if err != nil {
				return nil, err
			}
		}
	}

	// Add migration for enabled and attrs columns if they don't exist
	err = c.migrateUsersTable(ctx)
	if err != nil {
		return nil, err
	}

	if c.config.InitDb {
		err = c.initDB(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Synthesize the last-login events that back the usage-event feed. Cheap
	// and in-memory; only meaningful when --sync-user-last-login is set, but we
	// always seed so the feed is deterministic regardless of init-db timing.
	if err := c.seedLoginEvents(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.rawDB.Close()
}

func (c *Client) migrateUsersTable(ctx context.Context) error {
	// Check if the enabled and attrs columns exist
	query := "PRAGMA table_info(users)"
	rows, err := c.rawDB.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasEnabledColumn := false
	hasAttrsColumn := false
	hasAccountTypeColumn := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return err
		}
		switch name {
		case "enabled":
			hasEnabledColumn = true
		case "attrs":
			hasAttrsColumn = true
		case "account_type":
			hasAccountTypeColumn = true
		}
	}

	// If the enabled column doesn't exist, add it
	if !hasEnabledColumn {
		alterQuery := "ALTER TABLE users ADD COLUMN enabled BOOLEAN DEFAULT 1"
		_, err = c.rawDB.ExecContext(ctx, alterQuery)
		if err != nil {
			return err
		}
	}

	// If the attrs column doesn't exist, add it
	if !hasAttrsColumn {
		alterQuery := "ALTER TABLE users ADD COLUMN attrs BLOB"
		_, err = c.rawDB.ExecContext(ctx, alterQuery)
		if err != nil {
			return err
		}
	}

	// If the account_type column doesn't exist, add it (defaults to human).
	if !hasAccountTypeColumn {
		alterQuery := "ALTER TABLE users ADD COLUMN account_type TEXT NOT NULL DEFAULT 'human'"
		_, err = c.rawDB.ExecContext(ctx, alterQuery)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) validateDB() error {
	// Check if the database is already initialized
	if c.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

func (c *Client) initDB(ctx context.Context) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	l := ctxzap.Extract(ctx)
	l.Info("Initializing database", zap.Any("config", c.config))

	generator := &generator{
		config: c.config,
	}

	lastLogTime := time.Now()
	for {
		dbResource, ok := generator.Next()
		if !ok {
			break
		}

		if time.Since(lastLogTime) > 10*time.Second {
			l.Info("Inserting resource", zap.Any("resource", dbResource))
			lastLogTime = time.Now()
		}

		switch {
		case dbResource.User != nil:
			attrs, err := json.Marshal(dbResource.User.Attrs)
			if err != nil {
				return err
			}
			accountType := dbResource.User.AccountType
			if accountType == "" {
				accountType = AccountTypeHuman
			}
			row := goqu.Record{
				"id":           dbResource.User.Id,
				"name":         dbResource.User.Name,
				"email":        dbResource.User.Email,
				"enabled":      dbResource.User.Enabled,
				"account_type": accountType,
				"attrs":        attrs,
				"created_at":   dbResource.User.CreatedAt,
				"updated_at":   dbResource.User.UpdatedAt,
			}
			baseUserQ := c.db.Insert(users.Name()).Prepared(true)
			baseUserQ = baseUserQ.Rows(row)
			baseUserQ = baseUserQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseUserQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.Group != nil:
			row := goqu.Record{
				"id":         dbResource.Group.Id,
				"name":       dbResource.Group.Name,
				"admins":     strings.Join(dbResource.Group.Admins, ","),
				"members":    strings.Join(dbResource.Group.Members, ","),
				"created_at": dbResource.Group.CreatedAt,
				"updated_at": dbResource.Group.UpdatedAt,
			}
			baseGroupQ := c.db.Insert(groups.Name()).Prepared(true)
			baseGroupQ = baseGroupQ.Rows(row)
			baseGroupQ = baseGroupQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseGroupQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.Role != nil:
			row := goqu.Record{
				"id":                 dbResource.Role.Id,
				"name":               dbResource.Role.Name,
				"direct_assignments": strings.Join(dbResource.Role.DirectAssignments, ","),
				"group_assignments":  strings.Join(dbResource.Role.GroupAssignments, ","),
				"created_at":         dbResource.Role.CreatedAt,
				"updated_at":         dbResource.Role.UpdatedAt,
			}
			baseRoleQ := c.db.Insert(roles.Name()).Prepared(true)
			baseRoleQ = baseRoleQ.Rows(row)
			baseRoleQ = baseRoleQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseRoleQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.ScopedRole != nil:
			row := goqu.Record{
				"id":               dbResource.ScopedRole.Id,
				"project_id":       dbResource.ScopedRole.ProjectId,
				"role_id":          dbResource.ScopedRole.RoleId,
				"user_assignments": strings.Join(dbResource.ScopedRole.UserAssignments, ","),
				"created_at":       dbResource.ScopedRole.CreatedAt,
				"updated_at":       dbResource.ScopedRole.UpdatedAt,
			}
			baseScopedRoleQ := c.db.Insert(scopedRoles.Name()).Prepared(true)
			baseScopedRoleQ = baseScopedRoleQ.Rows(row)
			baseScopedRoleQ = baseScopedRoleQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseScopedRoleQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.Project != nil:
			row := goqu.Record{
				"id":                dbResource.Project.Id,
				"name":              dbResource.Project.Name,
				"owner":             dbResource.Project.Owner,
				"group_assignments": strings.Join(dbResource.Project.GroupAssignments, ","),
				"created_at":        dbResource.Project.CreatedAt,
				"updated_at":        dbResource.Project.UpdatedAt,
			}
			baseProjectQ := c.db.Insert(projects.Name()).Prepared(true)
			baseProjectQ = baseProjectQ.Rows(row)
			baseProjectQ = baseProjectQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseProjectQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.Password != nil:
			row := goqu.Record{
				"id":       dbResource.Password.Id,
				"user_id":  dbResource.Password.UserId,
				"password": dbResource.Password.Password,
			}
			basePasswordQ := c.db.Insert(passwords.Name()).Prepared(true)
			basePasswordQ = basePasswordQ.Rows(row)
			basePasswordQ = basePasswordQ.OnConflict(goqu.DoUpdate("id", row))
			query, args, err := basePasswordQ.ToSQL()
			if err != nil {
				return err
			}
			_, err = c.db.Exec(query, args...)
			if err != nil {
				return err
			}
		case dbResource.Secret != nil:
			if _, err := c.CreateSecret(ctx, dbResource.Secret); err != nil {
				return err
			}
		case dbResource.NHI != nil:
			row := goqu.Record{
				"id":         dbResource.NHI.Id,
				"name":       dbResource.NHI.Name,
				"kind":       dbResource.NHI.Kind,
				"nhi_type":   dbResource.NHI.NhiType,
				"nhi_detail": dbResource.NHI.NhiDetail,
				"created_at": dbResource.NHI.CreatedAt,
				"updated_at": dbResource.NHI.UpdatedAt,
			}
			baseNHIQ := c.db.Insert(nhis.Name()).Prepared(true).Rows(row).OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseNHIQ.ToSQL()
			if err != nil {
				return err
			}
			if _, err = c.db.Exec(query, args...); err != nil {
				return err
			}
		case dbResource.Agent != nil:
			profile, err := json.Marshal(dbResource.Agent.Profile)
			if err != nil {
				return err
			}
			row := goqu.Record{
				"id":          dbResource.Agent.Id,
				"name":        dbResource.Agent.Name,
				"status":      dbResource.Agent.Status,
				"identity_id": nullString(dbResource.Agent.IdentityID),
				"profile":     profile,
				"created_at":  dbResource.Agent.CreatedAt,
				"updated_at":  dbResource.Agent.UpdatedAt,
			}
			baseAgentQ := c.db.Insert(agents.Name()).Prepared(true).Rows(row).OnConflict(goqu.DoUpdate("id", row))
			query, args, err := baseAgentQ.ToSQL()
			if err != nil {
				return err
			}
			if _, err = c.db.Exec(query, args...); err != nil {
				return err
			}
		}
	}

	l.Info("Database initialized")

	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func (c *Client) rowToUser(_ context.Context, row scannable) (*User, error) {
	user := &User{}
	attrsBytes := []byte{}
	accountType := sql.NullString{}
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.Enabled, &accountType, &attrsBytes, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	user.AccountType = accountType.String
	if user.AccountType == "" {
		user.AccountType = AccountTypeHuman
	}
	if len(attrsBytes) > 0 {
		err = json.Unmarshal(attrsBytes, &user.Attrs)
		if err != nil {
			return nil, err
		}
	} else {
		user.Attrs = make(map[string]string)
	}
	return user, nil
}

// ListUsers returns all the users from the database.
func (c *Client) ListUsers(ctx context.Context, pToken *pagination.Token) ([]*User, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 500
	offset := 0
	if pToken != nil {
		limit = pToken.Size
		if limit <= 0 {
			limit = 500
		}
		if pToken.Token != "" {
			offset, err = strconv.Atoi(pToken.Token)
			if err != nil {
				return nil, "", err
			}
		}
	}
	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(users.Name()).Prepared(true)
	q = q.Select("id", "name", "email", "enabled", "account_type", "attrs", "created_at", "updated_at").
		Order(goqu.C("id").Asc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	usersList := []*User{}
	for rows.Next() {
		user, err := c.rowToUser(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		usersList = append(usersList, user)
	}

	nextPageToken := ""
	if len(usersList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return usersList, nextPageToken, nil
}

func (c *Client) ListUsersByUpdatedAt(ctx context.Context, updatedAt time.Time, sToken *pagination.StreamToken) ([]*User, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 50
	offset := 0
	if sToken != nil {
		limit = sToken.Size
		if sToken.Cursor != "" {
			offset, err = strconv.Atoi(sToken.Cursor)
			if err != nil {
				return nil, "", err
			}
		}
	}
	limit = min(max(limit, minEvents), maxEvents)

	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(users.Name()).Prepared(true).
		Select("id", "name", "email", "enabled", "account_type", "attrs", "created_at", "updated_at").
		Where(goqu.C("updated_at").Gt(updatedAt)).
		Order(goqu.C("updated_at").Desc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	usersList := []*User{}
	for rows.Next() {
		user, err := c.rowToUser(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		usersList = append(usersList, user)
	}

	nextPageToken := ""
	if len(usersList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return usersList, nextPageToken, nil
}

// GetUser returns the user requested if it exists, else returns an error.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(users.Name()).Prepared(true)
	q = q.Select("id", "name", "email", "enabled", "account_type", "attrs", "created_at", "updated_at")
	q = q.Where(goqu.C("id").Eq(userID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	user, err := c.rowToUser(ctx, row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	// Check if user exists
	_, err = c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Delete the user
	q := c.db.Delete(users.Name()).Prepared(true)
	q = q.Where(goqu.C("id").Eq(userID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateUser(ctx context.Context, name, email, password string) (*User, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		Id:          ksuid.New().String(),
		Name:        name,
		Email:       email,
		Enabled:     true, // Default to enabled
		AccountType: AccountTypeHuman,
		Attrs:       make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	attrs, err := json.Marshal(user.Attrs)
	if err != nil {
		return nil, err
	}

	q := c.db.Insert(users.Name()).Prepared(true)
	q = q.Rows(goqu.Record{
		"id":           user.Id,
		"name":         user.Name,
		"email":        user.Email,
		"enabled":      user.Enabled,
		"account_type": user.AccountType,
		"attrs":        attrs,
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	})

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	passwordQ := c.db.Update(passwords.Name()).Prepared(true)
	passwordQ = passwordQ.Set(goqu.Record{
		"password": password,
	})
	passwordQ = passwordQ.Where(goqu.C("user_id").Eq(user.Id))

	query, args, err = passwordQ.ToSQL()
	if err != nil {
		return nil, err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Client) UpdateUser(ctx context.Context, user *User) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	attrs, err := json.Marshal(user.Attrs)
	if err != nil {
		return err
	}

	now := time.Now()
	q := c.db.Update(users.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"name":       user.Name,
		"email":      user.Email,
		"enabled":    user.Enabled,
		"attrs":      attrs,
		"updated_at": now,
	})
	q = q.Where(goqu.C("id").Eq(user.Id))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ChangePassword(ctx context.Context, userID, password string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	// Check if user exists
	_, err = c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	q := c.db.Update(passwords.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"password": password,
	})
	q = q.Where(goqu.C("user_id").Eq(userID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) rowToGroup(_ context.Context, row scannable) (*Group, error) {
	group := &Group{}
	admins := ""
	members := ""
	err := row.Scan(&group.Id, &group.Name, &admins, &members, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if admins != "" {
		group.Admins = strings.Split(admins, ",")
	}
	if members != "" {
		group.Members = strings.Split(members, ",")
	}
	return group, nil
}

// ListGroups returns all the groups from the database.
func (c *Client) ListGroups(ctx context.Context) ([]*Group, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(groups.Name()).Prepared(true)
	q = q.Select("id", "name", "admins", "members", "created_at", "updated_at")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupsList := []*Group{}
	for rows.Next() {
		group, err := c.rowToGroup(ctx, rows)
		if err != nil {
			return nil, err
		}
		groupsList = append(groupsList, group)
	}

	return groupsList, nil
}

// GetGroup returns the group requested if it exists, else returns an error.
func (c *Client) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(groups.Name()).Prepared(true)
	q = q.Select("id", "name", "admins", "members", "created_at", "updated_at")
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToGroup(ctx, row)
}

func (c *Client) CreateGroup(ctx context.Context, groupID, groupName string, groupAdmins []string, groupMembers []string) (*Group, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	existingGroup, err := c.GetGroup(ctx, groupID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existingGroup != nil {
		return nil, errors.New("group already exists")
	}

	admins := ""
	if len(groupAdmins) > 0 {
		admins = strings.Join(groupAdmins, ",")
	}
	members := ""
	if len(groupMembers) > 0 {
		members = strings.Join(groupMembers, ",")
	}
	now := time.Now()
	row := goqu.Record{
		"id":         groupID,
		"name":       groupName,
		"admins":     admins,
		"members":    members,
		"created_at": now,
		"updated_at": now,
	}
	baseGroupQ := c.db.Insert(groups.Name()).Prepared(true)
	baseGroupQ = baseGroupQ.Rows(row)
	query, args, err := baseGroupQ.ToSQL()
	if err != nil {
		return nil, err
	}

	_, err = c.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	newGroup, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return newGroup, nil
}

func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	_, err = c.GetGroup(ctx, groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("group not found")
		}
		return err
	}

	q := c.db.Delete(groups.Name()).Prepared(true)
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GrantGroupMember(ctx context.Context, groupID, userID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check whether the user is already a member of the group
	if slices.Contains(group.Members, user.Id) {
		return nil
	}

	// Add the user to the group
	group.Members = append(group.Members, userID)
	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"members":    strings.Join(group.Members, ","),
		"updated_at": time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RevokeGroupMember(ctx context.Context, groupID, userID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	found := false
	for i, u := range group.Members {
		if u == user.Id {
			group.Members = append(group.Members[:i], group.Members[i+1:]...)
			found = true
		}
	}
	if !found {
		return nil
	}

	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"members":    strings.Join(group.Members, ","),
		"updated_at": time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GrantGroupAdmin(ctx context.Context, groupID, userID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// Check whether the user is already an admin of the group
	if slices.Contains(group.Admins, userID) {
		return nil
	}

	group.Admins = append(group.Admins, userID)

	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"admins":     strings.Join(group.Admins, ","),
		"updated_at": time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RevokeGroupAdmin(ctx context.Context, groupID, userID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	found := false
	for i, u := range group.Admins {
		if u == user.Id {
			group.Admins = append(group.Admins[:i], group.Admins[i+1:]...)
			found = true
		}
	}
	if !found {
		return nil
	}

	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"admins":     strings.Join(group.Admins, ","),
		"updated_at": time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) rowToRole(_ context.Context, row scannable) (*Role, error) {
	role := &Role{}
	directAssignments := ""
	groupAssignments := ""
	err := row.Scan(&role.Id, &role.Name, &directAssignments, &groupAssignments, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if directAssignments != "" {
		role.DirectAssignments = strings.Split(directAssignments, ",")
	}
	if groupAssignments != "" {
		role.GroupAssignments = strings.Split(groupAssignments, ",")
	}
	return role, nil
}

// ListRoles returns all the roles from the database.
func (c *Client) ListRoles(ctx context.Context) ([]*Role, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(roles.Name()).Prepared(true)
	q = q.Select("id", "name", "direct_assignments", "group_assignments", "created_at", "updated_at")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rolesList := []*Role{}
	for rows.Next() {
		role, err := c.rowToRole(ctx, rows)
		if err != nil {
			return nil, err
		}
		rolesList = append(rolesList, role)
	}

	return rolesList, nil
}

// GetRole returns the role requested if it exists, else returns an error.
func (c *Client) GetRole(ctx context.Context, roleID string) (*Role, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(roles.Name()).Prepared(true)
	q = q.Select("id", "name", "direct_assignments", "group_assignments", "created_at", "updated_at")
	q = q.Where(goqu.C("id").Eq(roleID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToRole(ctx, row)
}

func (c *Client) GrantRole(ctx context.Context, userID, roleID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	// Check if user exists
	_, err = c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check if role exists
	role, err := c.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	// Check if the user is already assigned the role
	if slices.Contains(role.DirectAssignments, userID) {
		return nil
	}

	// Update the role with the user
	role.DirectAssignments = append(role.DirectAssignments, userID)

	q := c.db.Update(roles.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"direct_assignments": strings.Join(role.DirectAssignments, ","),
		"updated_at":         time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(roleID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RevokeRole(ctx context.Context, userID, roleID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	// Check if user exists
	_, err = c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check if role exists
	role, err := c.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	found := false
	// If the user is assigned the role, remove the assignment
	for i, u := range role.DirectAssignments {
		if u == userID {
			role.DirectAssignments = append(role.DirectAssignments[:i], role.DirectAssignments[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return nil
	}

	q := c.db.Update(roles.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"direct_assignments": strings.Join(role.DirectAssignments, ","),
		"updated_at":         time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(roleID))

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) rowToScopedRole(_ context.Context, row scannable) (*ScopedRole, error) {
	scopedRole := &ScopedRole{}
	userAssignments := ""
	err := row.Scan(&scopedRole.Id, &scopedRole.ProjectId, &scopedRole.RoleId, &userAssignments, &scopedRole.CreatedAt, &scopedRole.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if userAssignments != "" {
		scopedRole.UserAssignments = strings.Split(userAssignments, ",")
	}
	return scopedRole, nil
}

func (c *Client) ListScopedRoles(ctx context.Context) ([]*ScopedRole, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(scopedRoles.Name()).Prepared(true)
	q = q.Select("id", "project_id", "role_id", "user_assignments", "created_at", "updated_at")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scopedRolesList := []*ScopedRole{}
	for rows.Next() {
		scopedRole, err := c.rowToScopedRole(ctx, rows)
		if err != nil {
			return nil, err
		}
		scopedRolesList = append(scopedRolesList, scopedRole)
	}

	return scopedRolesList, nil
}

func (c *Client) GetScopedRole(ctx context.Context, roleID, projectID string) (*ScopedRole, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(scopedRoles.Name()).Prepared(true)
	q = q.Select("id", "project_id", "role_id", "user_assignments", "created_at", "updated_at")
	q = q.Where(goqu.C("role_id").Eq(roleID), goqu.C("project_id").Eq(projectID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToScopedRole(ctx, row)
}

func (c *Client) CreateScopedRole(ctx context.Context, scopedRole *ScopedRole) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	now := time.Now()
	scopedRole.CreatedAt = now
	scopedRole.UpdatedAt = now
	row := goqu.Record{
		"id":               scopedRole.Id,
		"project_id":       scopedRole.ProjectId,
		"role_id":          scopedRole.RoleId,
		"user_assignments": strings.Join(scopedRole.UserAssignments, ","),
		"created_at":       scopedRole.CreatedAt,
		"updated_at":       scopedRole.UpdatedAt,
	}
	baseScopedRoleQ := c.db.Insert(scopedRoles.Name()).Prepared(true)
	baseScopedRoleQ = baseScopedRoleQ.Rows(row)
	baseScopedRoleQ = baseScopedRoleQ.OnConflict(goqu.DoUpdate("id", row))
	query, args, err := baseScopedRoleQ.ToSQL()
	if err != nil {
		return err
	}
	_, err = c.db.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateScopedRole(_ context.Context, scopedRole *ScopedRole) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	q := c.db.Update(scopedRoles.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"user_assignments": strings.Join(scopedRole.UserAssignments, ","),
		"updated_at":       time.Now(),
	})
	q = q.Where(goqu.C("id").Eq(scopedRole.Id))
	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	_, err = c.db.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteScopedRole(_ context.Context, roleID, projectID string) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	q := c.db.Delete(scopedRoles.Name()).Prepared(true)
	q = q.Where(goqu.C("role_id").Eq(roleID), goqu.C("project_id").Eq(projectID))
	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	_, err = c.db.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) rowToProject(_ context.Context, row scannable) (*Project, error) {
	project := &Project{}
	groupAssignments := ""
	err := row.Scan(&project.Id, &project.Name, &project.Owner, &groupAssignments, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if groupAssignments != "" {
		project.GroupAssignments = strings.Split(groupAssignments, ",")
	}
	return project, nil
}

// ListProjects returns all the projects from the database.
func (c *Client) ListProjects(ctx context.Context) ([]*Project, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(projects.Name()).Prepared(true)
	q = q.Select("id", "name", "owner", "group_assignments", "created_at", "updated_at")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projectsList := []*Project{}
	for rows.Next() {
		project, err := c.rowToProject(ctx, rows)
		if err != nil {
			return nil, err
		}
		projectsList = append(projectsList, project)
	}

	return projectsList, nil
}

// GetProject returns the project requested if it exists, else returns an error.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(projects.Name()).Prepared(true)
	q = q.Select("id", "name", "owner", "group_assignments", "created_at", "updated_at")
	q = q.Where(goqu.C("id").Eq(projectID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	return c.rowToProject(ctx, row)
}

func (c *Client) ListGroupsByUpdatedAt(ctx context.Context, updatedAt time.Time, sToken *pagination.StreamToken) ([]*Group, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 50
	offset := 0
	if sToken != nil {
		limit = sToken.Size
		if sToken.Cursor != "" {
			offset, err = strconv.Atoi(sToken.Cursor)
			if err != nil {
				return nil, "", err
			}
		}
	}
	limit = min(max(limit, minEvents), maxEvents)

	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(groups.Name()).Prepared(true).
		Select("id", "name", "admins", "members", "created_at", "updated_at").
		Where(goqu.C("updated_at").Gt(updatedAt)).
		Order(goqu.C("updated_at").Desc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	groupsList := []*Group{}
	for rows.Next() {
		group, err := c.rowToGroup(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		groupsList = append(groupsList, group)
	}

	nextPageToken := ""
	if len(groupsList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return groupsList, nextPageToken, nil
}

func (c *Client) ListRolesByUpdatedAt(ctx context.Context, updatedAt time.Time, sToken *pagination.StreamToken) ([]*Role, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 50
	offset := 0
	if sToken != nil {
		limit = sToken.Size
		if sToken.Cursor != "" {
			offset, err = strconv.Atoi(sToken.Cursor)
			if err != nil {
				return nil, "", err
			}
		}
	}
	limit = min(max(limit, minEvents), maxEvents)

	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(roles.Name()).Prepared(true).
		Select("id", "name", "direct_assignments", "group_assignments", "created_at", "updated_at").
		Where(goqu.C("updated_at").Gt(updatedAt)).
		Order(goqu.C("updated_at").Desc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	rolesList := []*Role{}
	for rows.Next() {
		role, err := c.rowToRole(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		rolesList = append(rolesList, role)
	}

	nextPageToken := ""
	if len(rolesList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return rolesList, nextPageToken, nil
}

func (c *Client) ListProjectsByUpdatedAt(ctx context.Context, updatedAt time.Time, sToken *pagination.StreamToken) ([]*Project, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 50
	offset := 0
	if sToken != nil {
		limit = sToken.Size
		if sToken.Cursor != "" {
			offset, err = strconv.Atoi(sToken.Cursor)
			if err != nil {
				return nil, "", err
			}
		}
	}
	limit = min(max(limit, minEvents), maxEvents)

	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(projects.Name()).Prepared(true).
		Select("id", "name", "owner", "group_assignments", "created_at", "updated_at").
		Where(goqu.C("updated_at").Gt(updatedAt)).
		Order(goqu.C("updated_at").Desc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	projectsList := []*Project{}
	for rows.Next() {
		project, err := c.rowToProject(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		projectsList = append(projectsList, project)
	}

	nextPageToken := ""
	if len(projectsList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return projectsList, nextPageToken, nil
}

func (c *Client) ListScopedRolesByUpdatedAt(ctx context.Context, updatedAt time.Time, sToken *pagination.StreamToken) ([]*ScopedRole, string, error) {
	err := c.validateDB()
	if err != nil {
		return nil, "", err
	}

	limit := 50
	offset := 0
	if sToken != nil {
		limit = sToken.Size
		if sToken.Cursor != "" {
			offset, err = strconv.Atoi(sToken.Cursor)
			if err != nil {
				return nil, "", err
			}
		}
	}
	limit = min(max(limit, minEvents), maxEvents)

	if offset < 0 {
		return nil, "", status.Errorf(codes.InvalidArgument, "offset cannot be negative")
	}

	q := c.db.From(scopedRoles.Name()).Prepared(true).
		Select("id", "project_id", "role_id", "user_assignments", "created_at", "updated_at").
		Where(goqu.C("updated_at").Gt(updatedAt)).
		Order(goqu.C("updated_at").Desc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, "", err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	scopedRolesList := []*ScopedRole{}
	for rows.Next() {
		scopedRole, err := c.rowToScopedRole(ctx, rows)
		if err != nil {
			return nil, "", err
		}
		scopedRolesList = append(scopedRolesList, scopedRole)
	}

	nextPageToken := ""
	if len(scopedRolesList) == limit {
		nextPageToken = strconv.Itoa(offset + limit)
	}

	return scopedRolesList, nextPageToken, nil
}
