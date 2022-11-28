package client

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
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

type Client struct{}

// ListUsers returns all the users from the database
func (c *Client) ListUsers(ctx context.Context) ([]*User, error) {
	log := ctxzap.Extract(ctx)

	log.Info("listing users", zap.Int("user_count", len(db.Users)))

	return db.Users, nil
}

// GetUser returns the user requested if it exists, else returns an error.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Id == userID {
			return u, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// ListGroups returns all the groups from the database
func (c *Client) ListGroups(ctx context.Context) ([]*Group, error) {
	log := ctxzap.Extract(ctx)

	log.Info("listing groups", zap.Int("group_count", len(db.Groups)))

	return db.Groups, nil
}

// GetGroup returns the group requested if it exists, else returns an error.
func (c *Client) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Id == groupID {
			return g, nil
		}
	}

	return nil, fmt.Errorf("group not found")
}

// ListRoles returns all the roles from the database
func (c *Client) ListRoles(ctx context.Context) ([]*Role, error) {
	log := ctxzap.Extract(ctx)

	log.Info("listing roles", zap.Int("role_count", len(db.Roles)))

	return db.Roles, nil
}

// GetRole returns the role requested if it exists, else returns an error.
func (c *Client) GetRole(ctx context.Context, roleID string) (*Role, error) {
	roles, err := c.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range roles {
		if r.Id == roleID {
			return r, nil
		}
	}

	return nil, fmt.Errorf("role not found")
}

// ListProjects returns all the projects from the database
func (c *Client) ListProjects(ctx context.Context) ([]*Project, error) {
	log := ctxzap.Extract(ctx)

	log.Info("listing projects", zap.Int("project_count", len(db.Roles)))

	return db.Projects, nil
}

// GetProject returns the project requested if it exists, else returns an error.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Id == projectID {
			return p, nil
		}
	}

	return nil, fmt.Errorf("project not found")
}
