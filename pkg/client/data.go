package client

import (
	"fmt"

	"github.com/conductorone/baton-demo/pkg/config"
)

type dbResource struct {
	User     *User
	Group    *Group
	Role     *Role
	Project  *Project
	Password *Password
}

type generator struct {
	config          *config.Demo
	currentUser     int
	currentPassword int
	currentGroup    int
	currentRole     int
	currentProject  int
}

func userId(i int) string {
	return fmt.Sprintf("user-%07d", i)
}

func (g *generator) Next() (*dbResource, bool) {
	db := &dbResource{}
	if g.currentGroup == 0 {
		// Make the Everyone group.
		groupAdmins := []string{userId(g.config.Users)}
		groupMembers := []string{}
		for i := 0; i < g.config.Users; i++ {
			groupMembers = append(groupMembers, userId(i))
		}

		db.Group = &Group{
			Id:      "group-everyone",
			Name:    "Everyone",
			Admins:  groupAdmins,
			Members: groupMembers,
		}
		g.currentGroup++
		return db, true
	}
	if g.currentGroup < g.config.Groups {
		// Split the users evenly into all groups.
		groupAdmins := []string{}
		groupMembers := []string{}
		for i := 0; i < g.config.Users; i++ {
			if i%g.config.Groups == g.currentGroup {
				if i%2 == 0 {
					groupAdmins = append(groupAdmins, userId(i))
				} else {
					groupMembers = append(groupMembers, userId(i))
				}
			}
		}
		db.Group = &Group{
			Id:      fmt.Sprintf("group-%07d", g.currentGroup),
			Name:    fmt.Sprintf("Group %07d", g.currentGroup),
			Admins:  groupAdmins,
			Members: groupMembers,
		}
		g.currentGroup++
		return db, true
	}
	if g.currentProject < g.config.Projects {
		db.Project = &Project{
			Id:    fmt.Sprintf("project-%07d", g.currentProject),
			Name:  fmt.Sprintf("Project %07d", g.currentProject),
			Owner: userId(g.currentProject % g.config.Users),
			GroupAssignments: []string{
				fmt.Sprintf("group-%07d", g.currentProject%g.config.Groups),
			},
		}
		g.currentProject++
		return db, true
	}
	if g.currentRole < g.config.Roles {
		db.Role = &Role{
			Id:   fmt.Sprintf("role-%07d", g.currentRole),
			Name: fmt.Sprintf("Role %07d", g.currentRole),
			DirectAssignments: []string{
				userId(g.currentRole % g.config.Users),
			},
			GroupAssignments: []string{
				fmt.Sprintf("group-%07d", g.currentRole%g.config.Groups),
			},
		}
		g.currentRole++
		return db, true
	}
	if g.currentUser < g.config.Users {
		db.User = &User{
			Id:    userId(g.currentUser),
			Name:  fmt.Sprintf("User %07d", g.currentUser),
			Email: fmt.Sprintf("user-%d@example.com", g.currentUser),
		}
		g.currentUser++
		return db, true
	}
	if g.currentPassword < g.config.Users {
		db.Password = &Password{
			Id:       fmt.Sprintf("password-%07d", g.currentPassword),
			Password: "password",
			UserId:   userId(g.currentPassword),
		}
		g.currentPassword++
		return db, true
	}

	return nil, false
}

var allTableDescriptors = []tableDescriptor{
	users,
	groups,
	roles,
	projects,
	passwords,
}

type tableDescriptor interface {
	Name() string
	Schema() (string, []any)
}

var users = (*usersTable)(nil)

type usersTable struct{}

func (t *usersTable) Name() string {
	return "users"
}

func (t *usersTable) Schema() (string, []any) {
	return "CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, email TEXT)", []any{}
}

var groups = (*groupsTable)(nil)

type groupsTable struct{}

func (t *groupsTable) Name() string {
	return "groups"
}

func (t *groupsTable) Schema() (string, []any) {
	return "CREATE TABLE IF NOT EXISTS groups (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, admins TEXT NOT NULL, members TEXT NOT NULL)", []any{}
}

var roles = (*rolesTable)(nil)

type rolesTable struct{}

func (t *rolesTable) Name() string {
	return "roles"
}

func (t *rolesTable) Schema() (string, []any) {
	return "CREATE TABLE IF NOT EXISTS roles (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, direct_assignments TEXT NOT NULL, group_assignments TEXT NOT NULL)", []any{}
}

var projects = (*projectsTable)(nil)

type projectsTable struct{}

func (t *projectsTable) Name() string {
	return "projects"
}

func (t *projectsTable) Schema() (string, []any) {
	return "CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner TEXT NOT NULL, group_assignments TEXT NOT NULL)", []any{}
}

var passwords = (*passwordsTable)(nil)

type passwordsTable struct{}

func (t *passwordsTable) Name() string {
	return "passwords"
}

func (t *passwordsTable) Schema() (string, []any) {
	return "CREATE TABLE IF NOT EXISTS passwords (id TEXT PRIMARY KEY, password TEXT NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id))", []any{}
}
