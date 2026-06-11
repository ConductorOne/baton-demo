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
	Secret     *Secret
	NHI        *NHI
	Agent      *Agent
}

func (r *dbResource) String() string {
	switch {
	case r.User != nil:
		return fmt.Sprintf("User: id %s name '%s' email '%s' type %s", r.User.Id, r.User.Name, r.User.Email, r.User.AccountType)
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
	case r.Secret != nil:
		return fmt.Sprintf("Secret: id %s type %s identity %s", r.Secret.Id, r.Secret.CredentialType, r.Secret.IdentityID)
	case r.NHI != nil:
		return fmt.Sprintf("NHI: id %s kind %s type %s", r.NHI.Id, r.NHI.Kind, r.NHI.NhiType)
	case r.Agent != nil:
		return fmt.Sprintf("Agent: id %s status %s identity %s", r.Agent.Id, r.Agent.Status, r.Agent.IdentityID)
	}
	return "Unknown"
}

type generator struct {
	config               *config.Demo
	currentUser          int
	currentPassword      int
	currentGroup         int
	currentRole          int
	currentScopedRole    int
	currentProject       int
	currentServiceAcct   int
	currentSystemAcct    int
	currentSecret        int
	currentUnownedSecret int
	currentNHIApp        int
	currentAssumableRole int
	currentAgent         int
}

func userId(i int) string {
	return fmt.Sprintf("user-%07d", i)
}

func serviceAccountId(i int) string {
	return fmt.Sprintf("service-account-%07d", i)
}

func systemAccountId(i int) string {
	return fmt.Sprintf("system-account-%07d", i)
}

func secretId(i int) string {
	return fmt.Sprintf("secret-%07d", i)
}

func unownedSecretId(i int) string {
	return fmt.Sprintf("secret-unowned-%07d", i)
}

func nhiAppId(i int) string {
	return fmt.Sprintf("nhi-app-%07d", i)
}

func assumableRoleId(i int) string {
	return fmt.Sprintf("nhi-role-%07d", i)
}

func agentId(i int) string {
	return fmt.Sprintf("agent-%07d", i)
}

// secretCredentialType cycles credential types so the estate exercises every
// SecretTrait credential_type and a representative platform detail string.
func secretCredentialType(i int) (string, string) {
	switch i % 3 {
	case 0:
		return CredentialTypeStaticSecret, "aws_access_key"
	case 1:
		return CredentialTypeAsymmetricKey, "ssh_key"
	default:
		return CredentialTypeCertificate, "x509"
	}
}

// serviceAccountIdentityFor returns the id of the service account that owns the
// i-th owned resource (secret or agent), or "" when no service accounts exist.
func (g *generator) serviceAccountIdentityFor(i int) string {
	if g.config.ServiceAccounts <= 0 {
		return ""
	}
	return serviceAccountId(i % g.config.ServiceAccounts)
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
				Id:        "group-everyone",
				Name:      "Everyone",
				Admins:    groupAdmins,
				Members:   groupMembers,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
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
				Id:        fmt.Sprintf("group-%07d", g.currentGroup),
				Name:      fmt.Sprintf("Group %07d", g.currentGroup),
				Admins:    groupAdmins,
				Members:   groupMembers,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
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
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
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
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
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
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
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
				Id:          userId(g.currentUser),
				Name:        userFullName,
				Email:       userEmail,
				Enabled:     true, // Default to enabled
				AccountType: AccountTypeHuman,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
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

	// K2 — service-account users (account_type SERVICE).
	if g.currentServiceAcct < g.config.ServiceAccounts {
		name := fmt.Sprintf("Service Account %07d", g.currentServiceAcct)
		id := serviceAccountId(g.currentServiceAcct)
		db := &dbResource{
			User: &User{
				Id:          id,
				Name:        name,
				Email:       fmt.Sprintf("%s@svc.example.com", id),
				Enabled:     true,
				AccountType: AccountTypeService,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Attrs: map[string]string{
					"full_name": name,
				},
			},
		}
		g.currentServiceAcct++
		return db, true
	}

	// K2 — system users (account_type SYSTEM).
	if g.currentSystemAcct < g.config.SystemAccounts {
		name := fmt.Sprintf("System Account %07d", g.currentSystemAcct)
		id := systemAccountId(g.currentSystemAcct)
		db := &dbResource{
			User: &User{
				Id:          id,
				Name:        name,
				Email:       fmt.Sprintf("%s@system.example.com", id),
				Enabled:     true,
				AccountType: AccountTypeSystem,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Attrs: map[string]string{
					"full_name": name,
				},
			},
		}
		g.currentSystemAcct++
		return db, true
	}

	// K1 — secrets owned by a service account (identity_id back-ref). Types
	// cycle and a deterministic subset carries expires_at / last_used_at so the
	// estate exercises expiry and staleness detectors.
	if g.currentSecret < g.config.Secrets {
		i := g.currentSecret
		credType, detail := secretCredentialType(i)
		now := time.Now()
		created := now.Add(-time.Duration(30+i) * 24 * time.Hour)
		secret := &Secret{
			Id:               secretId(i),
			Name:             fmt.Sprintf("Secret %07d", i),
			CredentialType:   credType,
			CredentialDetail: detail,
			IdentityID:       g.serviceAccountIdentityFor(i),
			CreatedAt:        created,
			UpdatedAt:        created,
		}
		// Half expire in the future, the rest are already expired.
		switch i % 4 {
		case 0:
			exp := now.Add(time.Duration(60+i) * 24 * time.Hour)
			secret.ExpiresAt = &exp
		case 1:
			exp := now.Add(-time.Duration(5+i) * 24 * time.Hour)
			secret.ExpiresAt = &exp
		}
		// Two thirds have been used; the remainder are never-used (a signal).
		if i%3 != 2 {
			used := now.Add(-time.Duration(i+1) * time.Hour)
			secret.LastUsedAt = &used
		}
		g.currentSecret++
		return &dbResource{Secret: secret}, true
	}

	// K1 — unowned secrets (no identity_id back-ref).
	if g.currentUnownedSecret < g.config.UnownedSecrets {
		i := g.currentUnownedSecret
		credType, detail := secretCredentialType(i)
		created := time.Now().Add(-time.Duration(90+i) * 24 * time.Hour)
		secret := &Secret{
			Id:               unownedSecretId(i),
			Name:             fmt.Sprintf("Unowned Secret %07d", i),
			CredentialType:   credType,
			CredentialDetail: detail,
			CreatedAt:        created,
			UpdatedAt:        created,
		}
		g.currentUnownedSecret++
		return &dbResource{Secret: secret}, true
	}

	// K3 — non-human-identity apps (TRAIT_APP). Alternate app-registration and
	// managed-identity.
	if g.currentNHIApp < g.config.NhiApps {
		i := g.currentNHIApp
		nhiType := NHITypeAppRegistration
		detail := "oauth_app"
		if i%2 == 1 {
			nhiType = NHITypeManagedIdentity
			detail = "azure_managed_identity"
		}
		db := &dbResource{
			NHI: &NHI{
				Id:        nhiAppId(i),
				Name:      fmt.Sprintf("NHI App %07d", i),
				Kind:      NHIKindApp,
				NhiType:   nhiType,
				NhiDetail: detail,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.currentNHIApp++
		return db, true
	}

	// K3 — assumable-role non-human identities (TRAIT_ROLE).
	if g.currentAssumableRole < g.config.AssumableRoles {
		i := g.currentAssumableRole
		db := &dbResource{
			NHI: &NHI{
				Id:        assumableRoleId(i),
				Name:      fmt.Sprintf("Assumable Role %07d", i),
				Kind:      NHIKindRole,
				NhiType:   NHITypeAssumableRole,
				NhiDetail: "aws_iam_role",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.currentAssumableRole++
		return db, true
	}

	// Agents (TRAIT_AGENT) each backed by a service account. Status cycles.
	if g.currentAgent < g.config.Agents {
		i := g.currentAgent
		status := AgentStatusReady
		switch i % 3 {
		case 1:
			status = AgentStatusDisabled
		case 2:
			status = AgentStatusDeleted
		}
		db := &dbResource{
			Agent: &Agent{
				Id:         agentId(i),
				Name:       fmt.Sprintf("Agent %07d", i),
				Status:     status,
				IdentityID: g.serviceAccountIdentityFor(i),
				Profile: map[string]string{
					"framework": "demo-agent-runtime",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.currentAgent++
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
	secrets,
	nhis,
	agents,
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
			account_type TEXT NOT NULL DEFAULT 'human',
			attrs BLOB,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			admins TEXT NOT NULL,
			members TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}, []any{}
}

var roles = (*rolesTable)(nil)

type rolesTable struct{}

func (t *rolesTable) Name() string {
	return "roles"
}

func (t *rolesTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			direct_assignments TEXT NOT NULL,
			group_assignments TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
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
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			owner TEXT NOT NULL,
			group_assignments TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
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

var secrets = (*secretsTable)(nil)

type secretsTable struct{}

func (t *secretsTable) Name() string {
	return "secrets"
}

func (t *secretsTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			credential_type TEXT NOT NULL,
			credential_detail TEXT,
			identity_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			last_used_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		"CREATE INDEX IF NOT EXISTS idx_secrets_identity_id ON secrets (identity_id);",
	}, []any{}
}

var nhis = (*nhisTable)(nil)

type nhisTable struct{}

func (t *nhisTable) Name() string {
	return "nhis"
}

func (t *nhisTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS nhis (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			nhi_type TEXT NOT NULL,
			nhi_detail TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		"CREATE INDEX IF NOT EXISTS idx_nhis_kind ON nhis (kind);",
	}, []any{}
}

var agents = (*agentsTable)(nil)

type agentsTable struct{}

func (t *agentsTable) Name() string {
	return "agents"
}

func (t *agentsTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			identity_id TEXT,
			profile BLOB,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}, []any{}
}
