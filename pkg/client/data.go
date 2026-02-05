package client

import (
	"fmt"
	"time"

	"github.com/conductorone/baton-demo/pkg/config"
)

type dbResource struct {
	User       *User
	Group      *Group
	Role       *Role
	ScopedRole *ScopedRole
	Project    *Project
	Password   *Password
}

func (r *dbResource) String() string {
	switch {
	case r.User != nil:
		return fmt.Sprintf("User: id %s name '%s' email '%s'", r.User.Id, r.User.Name, r.User.Email)
	case r.Group != nil:
		return fmt.Sprintf("Group: id %s name '%s' %d admins %d members", r.Group.Id, r.Group.Name, len(r.Group.Admins), len(r.Group.Members))
	case r.Role != nil:
		return fmt.Sprintf("Role: id %s name '%s' %d users %d groups", r.Role.Id, r.Role.Name, len(r.Role.DirectAssignments), len(r.Role.GroupAssignments))
	case r.ScopedRole != nil:
		return fmt.Sprintf("ScopedRole: id %s project %s role %s", r.ScopedRole.Id, r.ScopedRole.ProjectId, r.ScopedRole.RoleId)
	case r.Project != nil:
		return fmt.Sprintf("Project: id %s name '%s' owner %s %d groups", r.Project.Id, r.Project.Name, r.Project.Owner, len(r.Project.GroupAssignments))
	case r.Password != nil:
		return fmt.Sprintf("Password: id %s userid %s", r.Password.Id, r.Password.UserId)
	}
	return "Unknown"
}

type generator struct {
	config            *config.Demo
	currentUser       int
	currentPassword   int
	currentGroup      int
	currentRole       int
	currentScopedRole int
	currentProject    int
}

func userId(i int) string {
	return fmt.Sprintf("user-%07d", i)
}

func (g *generator) Next() (*dbResource, bool) {
	if g.currentGroup == g.config.Groups {
		// Make the Everyone group.
		groupAdmins := []string{userId(0)}
		groupMembers := []string{}
		for i := 0; i < g.config.Users; i++ {
			groupMembers = append(groupMembers, userId(i))
		}

		db := &dbResource{
			Group: &Group{
				Id:      "group-everyone",
				Name:    "Everyone",
				Admins:  groupAdmins,
				Members: groupMembers,
			},
		}
		g.currentGroup++
		return db, true
	}
	if g.currentGroup < g.config.Groups {
		groupAdmins := []string{}
		groupMembers := []string{}
		usersPerGroup := 20 // Add 5% of users to each group
		for i := 0; i < g.config.Users; i++ {
			if i%usersPerGroup == 0 {
				groupMembers = append(groupMembers, userId(i))
				if i%(usersPerGroup*10) == 0 {
					groupAdmins = append(groupAdmins, userId(i))
				}
			}
		}
		db := &dbResource{
			Group: &Group{
				Id:      fmt.Sprintf("group-%07d", g.currentGroup),
				Name:    fmt.Sprintf("Group %07d", g.currentGroup),
				Admins:  groupAdmins,
				Members: groupMembers,
			},
		}
		g.currentGroup++
		return db, true
	}
	if g.currentProject < g.config.Projects {
		db := &dbResource{
			Project: &Project{
				Id:    fmt.Sprintf("project-%07d", g.currentProject),
				Name:  fmt.Sprintf("Project %07d", g.currentProject),
				Owner: userId(g.currentProject % g.config.Users),
				GroupAssignments: []string{
					fmt.Sprintf("group-%07d", g.currentProject%g.config.Groups),
					fmt.Sprintf("group-%07d", (g.currentProject*10)%g.config.Groups),
				},
			},
		}
		g.currentProject++
		return db, true
	}
	if g.currentRole < g.config.Roles {
		directAssignments := []string{}
		if g.config.Users > 0 {
			directAssignments = append(directAssignments, userId(g.currentRole%g.config.Users))
			directAssignments = append(directAssignments, userId((g.currentRole*10)%g.config.Users))
		}
		groupAssignments := []string{}
		if g.config.Groups > 5 {
			groupAssignments = append(groupAssignments, fmt.Sprintf("group-%07d", g.currentRole%g.config.Groups))
			groupAssignments = append(groupAssignments, fmt.Sprintf("group-%07d", (g.currentRole*10)%g.config.Groups))
		}
		db := &dbResource{
			Role: &Role{
				Id:                fmt.Sprintf("role-%07d", g.currentRole),
				Name:              fmt.Sprintf("Role %07d", g.currentRole),
				DirectAssignments: directAssignments,
				GroupAssignments:  groupAssignments,
			},
		}
		g.currentRole++
		return db, true
	}
	if g.currentScopedRole < g.config.ScopedRoles {
		userAssignments := []string{}
		if g.config.Users > 0 {
			userAssignments = append(userAssignments, userId(g.currentScopedRole%g.config.Users))
			userAssignments = append(userAssignments, userId((g.currentScopedRole*5)%g.config.Users))
		}
		var db *dbResource
		if g.config.Projects > 0 && g.config.Roles > 0 {
			db = &dbResource{
				ScopedRole: &ScopedRole{
					Id:              fmt.Sprintf("scoped-role-%07d", g.currentScopedRole),
					ProjectId:       fmt.Sprintf("project-%07d", g.currentScopedRole%g.config.Projects),
					RoleId:          fmt.Sprintf("role-%07d", g.currentScopedRole%g.config.Roles),
					UserAssignments: userAssignments,
				},
			}
		}
		g.currentScopedRole++
		return db, true
	}
	if g.currentUser < g.config.Users {
		userFullName := fmt.Sprintf("User %07d", g.currentUser)
		userEmail := fmt.Sprintf("user-%07d@example.com", g.currentUser)
		db := &dbResource{
			User: &User{
				Id:        userId(g.currentUser),
				Name:      userFullName,
				Email:     userEmail,
				Enabled:   true, // Default to enabled
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Attrs: map[string]string{
					"full_name": userFullName,
					"email":     userEmail,
				},
			},
		}
		g.currentUser++
		return db, true
	}
	if g.currentPassword < g.config.Users {
		db := &dbResource{
			Password: &Password{
				Id:       fmt.Sprintf("password-%07d", g.currentPassword),
				Password: "password",
				UserId:   userId(g.currentPassword),
			},
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
	scopedRoles,
	projects,
	passwords,
}

type tableDescriptor interface {
	Name() string
	Schema() ([]string, []any)
}

var users = (*usersTable)(nil)

type usersTable struct{}

func (t *usersTable) Name() string {
	return "users"
}

func (t *usersTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY, 
			name TEXT NOT NULL UNIQUE, 
			email TEXT, 
			enabled BOOLEAN DEFAULT 1, 
			attrs BLOB, 
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, 
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}, []any{}
}

var groups = (*groupsTable)(nil)

type groupsTable struct{}

func (t *groupsTable) Name() string {
	return "groups"
}

func (t *groupsTable) Schema() ([]string, []any) {
	return []string{
		"CREATE TABLE IF NOT EXISTS groups (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, admins TEXT NOT NULL, members TEXT NOT NULL)",
	}, []any{}
}

var roles = (*rolesTable)(nil)

type rolesTable struct{}

func (t *rolesTable) Name() string {
	return "roles"
}

func (t *rolesTable) Schema() ([]string, []any) {
	return []string{
		"CREATE TABLE IF NOT EXISTS roles (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, direct_assignments TEXT NOT NULL, group_assignments TEXT NOT NULL)",
	}, []any{}
}

var scopedRoles = (*scopedRolesTable)(nil)

type scopedRolesTable struct{}

func (t *scopedRolesTable) Name() string {
	return "scoped_roles"
}

func (t *scopedRolesTable) Schema() ([]string, []any) {
	statement := []string{`
	CREATE TABLE IF NOT EXISTS scoped_roles (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		role_id TEXT NOT NULL,
		user_assignments TEXT NOT NULL,
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(role_id) REFERENCES roles(id)
	);`,
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_scoped_roles_project_id_role_id ON scoped_roles (project_id, role_id);",
	}
	return statement, []any{}
}

var projects = (*projectsTable)(nil)

type projectsTable struct{}

func (t *projectsTable) Name() string {
	return "projects"
}

func (t *projectsTable) Schema() ([]string, []any) {
	return []string{
		"CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner TEXT NOT NULL, group_assignments TEXT NOT NULL)",
	}, []any{}
}

var passwords = (*passwordsTable)(nil)

type passwordsTable struct{}

func (t *passwordsTable) Name() string {
	return "passwords"
}

func (t *passwordsTable) Schema() ([]string, []any) {
	return []string{
		"CREATE TABLE IF NOT EXISTS passwords (id TEXT PRIMARY KEY, password TEXT NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id))",
	}, []any{}
}
