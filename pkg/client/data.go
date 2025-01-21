package client

type database struct {
	Users     []*User
	Groups    []*Group
	Roles     []*Role
	Projects  []*Project
	Passwords map[string]string
}

func generateDB() *database {
	db := &database{}

	db.Users = []*User{
		{
			Id:    "2IC0Wn5oRQqVVn3COFl1O1zSzV6",
			Name:  "Alice",
			Email: "alice@example.com",
		},
		{
			Id:    "2IC0WoNfqUPT7mgO4FOaViIxBrR",
			Name:  "Bob",
			Email: "bob@example.com",
		},
		{
			Id:    "2IC0Wo34fcTerFEgWmyffXmfrW8",
			Name:  "Carol",
			Email: "carol@example.com",
		},
		{
			Id:    "2IC0Wn7DaxV1xqDpdg7jJRiPtCp",
			Name:  "Dan",
			Email: "dan@example.com",
		},
		{
			Id:    "2IC0WoaHVvl2GIQppXQH0flK1yJ",
			Name:  "Frank",
			Email: "frank@example.com",
		},
	}

	db.Passwords = map[string]string{
		"2IC0Wn5oRQqVVn3COFl1O1zSzV6": "password",
		"2IC0WoNfqUPT7mgO4FOaViIxBrR": "password",
		"2IC0Wo34fcTerFEgWmyffXmfrW8": "password",
		"2IC0Wn7DaxV1xqDpdg7jJRiPtCp": "password",
		"2IC0WoaHVvl2GIQppXQH0flK1yJ": "password",
	}

	db.Groups = []*Group{
		{
			Id:   "2IC0WmAPkihbFdZhEPsch5N5WNO",
			Name: "Engineers",
			Admins: []string{
				"2IC0Wo34fcTerFEgWmyffXmfrW8", // Carol
			},
			Members: []string{
				"2IC0Wn5oRQqVVn3COFl1O1zSzV6", // Alice
				"2IC0WoNfqUPT7mgO4FOaViIxBrR", // Bob
			},
		},
		{
			Id:   "2IC0WjepYDBsRp6b7cqrumGsVGt",
			Name: "Sales",
			Admins: []string{
				"2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			},
			Members: []string{
				"2IC0Wn7DaxV1xqDpdg7jJRiPtCp", // Dan
			},
		},
	}

	db.Roles = []*Role{
		{
			Id:   "2IC0WmaHecJdzo5jYnQiTh2BVlB",
			Name: "Editor",
			DirectAssignments: []string{
				"2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			},
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
			},
		},
		{
			Id:                "2IC0WkRTFmsXH4P9TjiQnd29XMT",
			Name:              "Reader",
			DirectAssignments: []string{}, // No direct assignments
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
	}

	db.Projects = []*Project{
		{
			Id:    "2IC0WqENS0dCRHiJ0YvPAidl0D5",
			Name:  "Product X",
			Owner: "2IC0WoNfqUPT7mgO4FOaViIxBrR", // Bob
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
		{
			Id:    "2IC11NXgAkNrKRk9nukbPRRKMhI",
			Name:  "Sales",
			Owner: "2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			GroupAssignments: []string{
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
	}

	return db
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
	Schema() (string, []interface{})
}

var users = (*usersTable)(nil)

type usersTable struct{}

func (t *usersTable) Name() string {
	return "users"
}

func (t *usersTable) Schema() (string, []interface{}) {
	return "CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, email TEXT)", []interface{}{}
}

var groups = (*groupsTable)(nil)

type groupsTable struct{}

func (t *groupsTable) Name() string {
	return "groups"
}

func (t *groupsTable) Schema() (string, []interface{}) {
	return "CREATE TABLE IF NOT EXISTS groups (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, admins TEXT NOT NULL, members TEXT NOT NULL)", []interface{}{}
}

var roles = (*rolesTable)(nil)

type rolesTable struct{}

func (t *rolesTable) Name() string {
	return "roles"
}

func (t *rolesTable) Schema() (string, []interface{}) {
	return "CREATE TABLE IF NOT EXISTS roles (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, direct_assignments TEXT NOT NULL, group_assignments TEXT NOT NULL)", []interface{}{}
}

var projects = (*projectsTable)(nil)

type projectsTable struct{}

func (t *projectsTable) Name() string {
	return "projects"
}

func (t *projectsTable) Schema() (string, []interface{}) {
	return "CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner TEXT NOT NULL, group_assignments TEXT NOT NULL)", []interface{}{}
}

var passwords = (*passwordsTable)(nil)

type passwordsTable struct{}

func (t *passwordsTable) Name() string {
	return "passwords"
}

func (t *passwordsTable) Schema() (string, []interface{}) {
	return "CREATE TABLE IF NOT EXISTS passwords (id TEXT PRIMARY KEY, password TEXT NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id))", []interface{}{}
}
