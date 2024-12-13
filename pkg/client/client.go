package client

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/segmentio/ksuid"

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
// Projects always have a single User as the owner, and can be assigned to Groups

type User struct {
	Id    string
	Name  string
	Email string
}

type Group struct {
	Id      string
	Name    string
	Admins  []string
	Members []string
}

type Role struct {
	Id                string
	Name              string
	DirectAssignments []string
	GroupAssignments  []string
}

type Project struct {
	Id               string
	Name             string
	Owner            string
	GroupAssignments []string
}

// Client is a simple example client. While this client would normally be responsible for communicating with an upstream.
// API, for this demo the client is only working with in-memory data.
type Client struct {
	db         *goqu.Database
	rawDB      *sql.DB
	dbFileName string
}

func NewClient(dbFileName string, initDB bool) (*Client, error) {
	c := &Client{}

	// Open the database file
	if dbFileName == "" {
		dbFileName = "baton-demo.db"
	}

	rawDB, err := sql.Open("sqlite", dbFileName)
	if err != nil {
		return nil, err
	}

	db := goqu.New("sqlite3", rawDB)
	c.dbFileName = dbFileName
	c.db = db
	c.rawDB = rawDB

	err = c.initDB(initDB)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.rawDB.Close()
}

func (c *Client) validateDB() error {
	// Check if the database is already initialized
	if c.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

func (c *Client) initDB(initDB bool) error {
	err := c.validateDB()
	if err != nil {
		return err
	}

	// ensure all schemas exist
	for _, t := range allTableDescriptors {
		query, args := t.Schema()

		_, err := c.db.Exec(query, args...)
		if err != nil {
			return err
		}
	}

	if initDB {
		seedData := generateDB()
		err = c.db.WithTx(func(tx *goqu.TxDatabase) error {
			baseUserQ := tx.Insert(users.Name()).Prepared(true)
			baseUserQ = baseUserQ.OnConflict(goqu.DoNothing())
			for _, user := range seedData.Users {
				query, args, err := baseUserQ.Rows(goqu.Record{
					"id":    user.Id,
					"name":  user.Name,
					"email": user.Email,
				}).ToSQL()
				if err != nil {
					return err
				}

				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}

			baseGroupQ := tx.Insert(groups.Name()).Prepared(true)
			baseGroupQ = baseGroupQ.OnConflict(goqu.DoNothing())
			for _, group := range seedData.Groups {
				query, args, err := baseGroupQ.Rows(goqu.Record{
					"id":      group.Id,
					"name":    group.Name,
					"admins":  strings.Join(group.Admins, ","),
					"members": strings.Join(group.Members, ","),
				}).ToSQL()
				if err != nil {
					return err
				}

				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}

			baseRoleQ := tx.Insert(roles.Name()).Prepared(true)
			baseRoleQ = baseRoleQ.OnConflict(goqu.DoNothing())
			for _, role := range seedData.Roles {
				query, args, err := baseRoleQ.Rows(goqu.Record{
					"id":                 role.Id,
					"name":               role.Name,
					"direct_assignments": strings.Join(role.DirectAssignments, ","),
					"group_assignments":  strings.Join(role.GroupAssignments, ","),
				}).ToSQL()
				if err != nil {
					return err
				}

				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}

			baseProjectQ := tx.Insert(projects.Name()).Prepared(true)
			baseProjectQ = baseProjectQ.OnConflict(goqu.DoNothing())
			for _, project := range seedData.Projects {
				query, args, err := baseProjectQ.Rows(goqu.Record{
					"id":                project.Id,
					"name":              project.Name,
					"owner":             project.Owner,
					"group_assignments": strings.Join(project.GroupAssignments, ","),
				}).ToSQL()
				if err != nil {
					return err
				}

				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}

			basePasswordQ := tx.Insert(passwords.Name()).Prepared(true)
			basePasswordQ = basePasswordQ.OnConflict(goqu.DoNothing())
			for userID, password := range seedData.Passwords {
				query, args, err := basePasswordQ.Rows(goqu.Record{
					"id":       ksuid.New().String(),
					"user_id":  userID,
					"password": password,
				}).ToSQL()
				if err != nil {
					return err
				}

				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// ListUsers returns all the users from the database.
func (c *Client) ListUsers(ctx context.Context) ([]*User, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(users.Name()).Prepared(true)
	q = q.Select("id", "name", "email")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	usersList := []*User{}
	for rows.Next() {
		user := &User{}
		err = rows.Scan(&user.Id, &user.Name, &user.Email)
		if err != nil {
			return nil, err
		}
		usersList = append(usersList, user)
	}

	return usersList, nil
}

// GetUser returns the user requested if it exists, else returns an error.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(users.Name()).Prepared(true)
	q = q.Select("id", "name", "email")
	q = q.Where(goqu.C("id").Eq(userID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	user := &User{}
	err = row.Scan(&user.Id, &user.Name, &user.Email)
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

	user := &User{
		Id:    ksuid.New().String(),
		Name:  name,
		Email: email,
	}

	q := c.db.Insert(users.Name()).Prepared(true)
	q = q.Rows(goqu.Record{
		"id":    user.Id,
		"name":  user.Name,
		"email": user.Email,
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

// ListGroups returns all the groups from the database.
func (c *Client) ListGroups(ctx context.Context) ([]*Group, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(groups.Name()).Prepared(true)
	q = q.Select("id", "name", "admins", "members")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	groupsList := []*Group{}
	for rows.Next() {
		group := &Group{}
		admins := ""
		members := ""
		err = rows.Scan(&group.Id, &group.Name, &admins, &members)
		if err != nil {
			return nil, err
		}

		group.Admins = strings.Split(admins, ",")
		group.Members = strings.Split(members, ",")
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
	q = q.Select("id", "name", "admins", "members")
	q = q.Where(goqu.C("id").Eq(groupID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	group := &Group{}
	admins := ""
	members := ""
	err = row.Scan(&group.Id, &group.Name, &admins, &members)
	if err != nil {
		return nil, err
	}

	group.Admins = strings.Split(admins, ",")
	group.Members = strings.Split(members, ",")

	return group, nil
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
	for _, u := range group.Members {
		if u == user.Id {
			return nil
		}
	}

	// Add the user to the group
	group.Members = append(group.Members, userID)
	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"members": strings.Join(group.Members, ","),
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
		"members": strings.Join(group.Members, ","),
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

	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check whether the user is already an admin of the group
	for _, u := range group.Admins {
		if u == user.Id {
			return nil
		}
	}

	group.Admins = append(group.Admins, userID)

	q := c.db.Update(groups.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"admins": strings.Join(group.Admins, ","),
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
		"admins": strings.Join(group.Admins, ","),
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

// ListRoles returns all the roles from the database.
func (c *Client) ListRoles(ctx context.Context) ([]*Role, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(roles.Name()).Prepared(true)
	q = q.Select("id", "name", "direct_assignments", "group_assignments")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	rolesList := []*Role{}
	for rows.Next() {
		role := &Role{}
		directAssignments := ""
		groupAssignments := ""
		err = rows.Scan(&role.Id, &role.Name, &directAssignments, &groupAssignments)
		if err != nil {
			return nil, err
		}

		role.DirectAssignments = strings.Split(directAssignments, ",")
		role.GroupAssignments = strings.Split(groupAssignments, ",")
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
	q = q.Select("id", "name", "direct_assignments", "group_assignments")
	q = q.Where(goqu.C("id").Eq(roleID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	role := &Role{}
	directAssignments := ""
	groupAssignments := ""
	err = row.Scan(&role.Id, &role.Name, &directAssignments, &groupAssignments)
	if err != nil {
		return nil, err
	}

	role.DirectAssignments = strings.Split(directAssignments, ",")
	role.GroupAssignments = strings.Split(groupAssignments, ",")

	return role, nil
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
	for _, u := range role.DirectAssignments {
		if u == userID {
			return nil
		}
	}

	// Update the role with the user
	role.DirectAssignments = append(role.DirectAssignments, userID)

	q := c.db.Update(roles.Name()).Prepared(true)
	q = q.Set(goqu.Record{
		"direct_assignments": strings.Join(role.DirectAssignments, ","),
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

// ListProjects returns all the projects from the database.
func (c *Client) ListProjects(ctx context.Context) ([]*Project, error) {
	err := c.validateDB()
	if err != nil {
		return nil, err
	}

	q := c.db.From(projects.Name()).Prepared(true)
	q = q.Select("id", "name", "owner", "group_assignments")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	projectsList := []*Project{}
	for rows.Next() {
		project := &Project{}
		groupAssignments := ""
		err = rows.Scan(&project.Id, &project.Name, &project.Owner, &groupAssignments)
		if err != nil {
			return nil, err
		}

		project.GroupAssignments = strings.Split(groupAssignments, ",")
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
	q = q.Select("id", "name", "owner", "group_assignments")
	q = q.Where(goqu.C("id").Eq(projectID))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	row := c.db.QueryRowContext(ctx, query, args...)
	project := &Project{}
	groupAssignments := ""
	err = row.Scan(&project.Id, &project.Name, &project.Owner, &groupAssignments)
	if err != nil {
		return nil, err
	}

	project.GroupAssignments = strings.Split(groupAssignments, ",")

	return project, nil
}
