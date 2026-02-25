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

// Department definitions with proportional weights.
type departmentDef struct {
	Name      string
	Weight    int // Cumulative percentage boundary (0-100)
	JobTitles []string
}

// departments defines the org structure. Weights are cumulative boundaries:
// Engineering=0-34, Sales=35-54, Marketing=55-66, Finance=67-76, HR=77-86, Product=87-99.
var departments = []departmentDef{
	{Name: "Engineering", Weight: 35, JobTitles: []string{
		"Software Engineer", "Senior Software Engineer", "Staff Engineer",
		"Engineering Manager", "QA Engineer", "DevOps Engineer", "Frontend Engineer",
	}},
	{Name: "Sales", Weight: 55, JobTitles: []string{
		"Account Executive", "Sales Development Rep", "Sales Manager",
		"Solutions Engineer", "Sales Operations Analyst",
	}},
	{Name: "Marketing", Weight: 67, JobTitles: []string{
		"Marketing Manager", "Content Writer", "Growth Analyst",
		"Brand Designer", "SEO Specialist",
	}},
	{Name: "Finance", Weight: 77, JobTitles: []string{
		"Financial Analyst", "Accountant", "Controller", "FP&A Analyst",
	}},
	{Name: "Human Resources", Weight: 87, JobTitles: []string{
		"HR Business Partner", "Recruiter", "Compensation Analyst",
		"HR Manager", "People Operations",
	}},
	{Name: "Product", Weight: 100, JobTitles: []string{
		"Product Manager", "Senior Product Manager", "Product Designer",
		"UX Researcher", "Technical Writer",
	}},
}

// appDef defines a SaaS application group with department-correlated access.
type appDef struct {
	Name         string
	DeptCoverage map[string]int // department name -> coverage percentage (0-100)
	NoisePct     int            // percentage of non-target dept users who also get access
}

// appGroups defines the SaaS applications and their department access patterns.
// Coverage percentages control what fraction of users in each department get the app.
// NoisePct controls cross-department "outlier" access.
var appGroups = []appDef{
	// Universal apps - everyone gets these
	{Name: "Google Workspace", DeptCoverage: map[string]int{
		"Engineering": 100, "Sales": 100, "Marketing": 100,
		"Finance": 100, "Human Resources": 100, "Product": 100,
	}},
	{Name: "Slack", DeptCoverage: map[string]int{
		"Engineering": 100, "Sales": 100, "Marketing": 100,
		"Finance": 100, "Human Resources": 100, "Product": 100,
	}},
	{Name: "Okta", DeptCoverage: map[string]int{
		"Engineering": 100, "Sales": 100, "Marketing": 100,
		"Finance": 100, "Human Resources": 100, "Product": 100,
	}},
	{Name: "1Password", DeptCoverage: map[string]int{
		"Engineering": 97, "Sales": 95, "Marketing": 95,
		"Finance": 96, "Human Resources": 95, "Product": 96,
	}},

	// Engineering tools
	{Name: "GitHub", DeptCoverage: map[string]int{"Engineering": 96, "Product": 35}, NoisePct: 3},
	{Name: "Jira", DeptCoverage: map[string]int{"Engineering": 93, "Product": 88, "Marketing": 25}, NoisePct: 4},
	{Name: "AWS Console", DeptCoverage: map[string]int{"Engineering": 88}, NoisePct: 2},
	{Name: "Datadog", DeptCoverage: map[string]int{"Engineering": 83}, NoisePct: 1},
	{Name: "PagerDuty", DeptCoverage: map[string]int{"Engineering": 74}, NoisePct: 1},
	{Name: "CircleCI", DeptCoverage: map[string]int{"Engineering": 68}, NoisePct: 1},

	// Sales tools
	{Name: "Salesforce", DeptCoverage: map[string]int{"Sales": 96}, NoisePct: 2},
	{Name: "HubSpot", DeptCoverage: map[string]int{"Sales": 91, "Marketing": 88}, NoisePct: 3},
	{Name: "Gong", DeptCoverage: map[string]int{"Sales": 87}, NoisePct: 2},
	{Name: "Outreach", DeptCoverage: map[string]int{"Sales": 78}, NoisePct: 1},
	{Name: "LinkedIn Sales Navigator", DeptCoverage: map[string]int{"Sales": 73}, NoisePct: 1},

	// Marketing tools
	{Name: "Google Analytics", DeptCoverage: map[string]int{"Marketing": 90, "Product": 82}, NoisePct: 3},
	{Name: "Figma", DeptCoverage: map[string]int{"Marketing": 91, "Product": 89, "Engineering": 22}, NoisePct: 3},
	{Name: "Canva", DeptCoverage: map[string]int{"Marketing": 83}, NoisePct: 2},
	{Name: "Mailchimp", DeptCoverage: map[string]int{"Marketing": 78}, NoisePct: 1},

	// Finance tools
	{Name: "NetSuite", DeptCoverage: map[string]int{"Finance": 96}, NoisePct: 1},
	{Name: "Expensify", DeptCoverage: map[string]int{"Finance": 92}, NoisePct: 2},
	{Name: "Stripe Dashboard", DeptCoverage: map[string]int{"Finance": 76, "Engineering": 28}, NoisePct: 2},
	{Name: "QuickBooks", DeptCoverage: map[string]int{"Finance": 69}, NoisePct: 1},

	// HR tools
	{Name: "Workday", DeptCoverage: map[string]int{"Human Resources": 97}, NoisePct: 1},
	{Name: "Greenhouse", DeptCoverage: map[string]int{"Human Resources": 93}, NoisePct: 2},
	{Name: "BambooHR", DeptCoverage: map[string]int{"Human Resources": 88}, NoisePct: 1},
	{Name: "Culture Amp", DeptCoverage: map[string]int{"Human Resources": 73}, NoisePct: 1},

	// Product tools
	{Name: "Amplitude", DeptCoverage: map[string]int{"Product": 89}, NoisePct: 2},
	{Name: "FullStory", DeptCoverage: map[string]int{"Product": 78}, NoisePct: 1},
	{Name: "Notion", DeptCoverage: map[string]int{"Product": 87, "Engineering": 72}, NoisePct: 4},
}

// Realistic first names for user generation.
var firstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda",
	"David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Christopher", "Karen", "Charles", "Lisa", "Daniel", "Nancy",
	"Matthew", "Betty", "Anthony", "Dorothy", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Carol", "Kevin", "Amanda", "Brian", "Melissa", "George", "Deborah",
	"Timothy", "Stephanie", "Ronald", "Rebecca", "Jason", "Sharon", "Edward", "Laura",
	"Jeffrey", "Cynthia", "Ryan", "Kathleen", "Jacob", "Amy", "Gary", "Angela",
	"Nicholas", "Shirley", "Eric", "Brenda", "Jonathan", "Emma", "Stephen", "Anna",
	"Larry", "Pamela", "Justin", "Nicole", "Scott", "Samantha", "Brandon", "Katherine",
	"Benjamin", "Christine", "Samuel", "Helen", "Raymond", "Debra", "Gregory", "Rachel",
	"Frank", "Carolyn", "Alexander", "Janet", "Patrick", "Catherine", "Jack", "Maria",
}

// Realistic last names for user generation.
var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
	"Thomas", "Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson",
	"White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker",
	"Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill",
	"Flores", "Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell",
	"Mitchell", "Carter", "Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz",
	"Parker", "Cruz", "Edwards", "Collins", "Reyes", "Stewart", "Morris", "Morales",
	"Murphy", "Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson",
	"Bailey", "Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward",
	"Richardson", "Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray",
	"Mendoza", "Ruiz", "Hughes", "Price", "Alvarez", "Castillo", "Sanders", "Patel",
}

// getDepartment returns the department and job title for a given user index.
func getDepartment(userIdx, totalUsers int) (string, string) {
	// Map user index to a percentage bucket (0-99)
	pct := (userIdx * 100) / totalUsers
	for _, dept := range departments {
		if pct < dept.Weight {
			titleIdx := userIdx % len(dept.JobTitles)
			return dept.Name, dept.JobTitles[titleIdx]
		}
	}
	// Fallback to last department
	last := departments[len(departments)-1]
	return last.Name, last.JobTitles[userIdx%len(last.JobTitles)]
}

// isManagerTitle returns true if the job title is a management role.
func isManagerTitle(title string) bool {
	switch title {
	case "Engineering Manager", "Sales Manager", "Marketing Manager",
		"HR Manager", "Controller", "Senior Product Manager":
		return true
	}
	return false
}

// getManagerEmail returns the email of the first manager in the same department.
// For managers themselves, returns the email of the next manager (or empty if they're the only one).
// totalUsers must match the value used during generation.
func getManagerEmail(userIdx, totalUsers int) string {
	dept, title := getDepartment(userIdx, totalUsers)
	isManager := isManagerTitle(title)

	// Scan all users to find the first manager in the same department.
	for i := 0; i < totalUsers; i++ {
		if i == userIdx {
			continue
		}
		d, t := getDepartment(i, totalUsers)
		if d != dept {
			continue
		}
		if !isManagerTitle(t) {
			continue
		}
		// Non-managers report to the first manager found.
		// Managers report to a different manager (not themselves).
		if !isManager || i != userIdx {
			first, last := getUserName(i)
			return fmt.Sprintf("%s.%s.%d@example.com", first, last, i)
		}
	}
	return ""
}

// getUserName returns a realistic first/last name for a user index.
func getUserName(userIdx int) (string, string) {
	first := firstNames[userIdx%len(firstNames)]
	// Offset last name index to avoid repetitive "James Smith, Mary Johnson" patterns
	last := lastNames[(userIdx*7+userIdx/len(firstNames))%len(lastNames)]
	return first, last
}

// shouldAssign is a deterministic hash-like function that decides group membership.
// Returns true if the user at userIdx should be a member of the group at groupIdx
// given the coverage percentage.
func shouldAssign(userIdx, groupIdx, coveragePct int) bool {
	if coveragePct >= 100 {
		return true
	}
	if coveragePct <= 0 {
		return false
	}
	// Knuth multiplicative hash for good distribution
	h := uint32(userIdx)*2654435761 + uint32(groupIdx)*2246822519
	// Mix bits for better uniformity
	h ^= h >> 16
	h *= 0x45d9f3b
	h ^= h >> 16
	return int(h%100) < coveragePct
}

type generator struct {
	config            *config.Demo
	currentUser       int
	currentPassword   int
	currentGroup      int
	currentRole       int
	currentScopedRole int
	currentProject    int
	groupsDone        bool // true after all app groups + Everyone group are emitted
}

func userId(i int) string {
	return fmt.Sprintf("user-%07d", i)
}

func groupId(i int) string {
	return fmt.Sprintf("group-%07d", i)
}

func (g *generator) totalAppGroups() int {
	return len(appGroups)
}

func (g *generator) Next() (*dbResource, bool) {
	// Phase 1: Generate app-based groups
	if !g.groupsDone {
		if g.currentGroup < g.totalAppGroups() {
			app := appGroups[g.currentGroup]
			members := []string{}
			admins := []string{}

			for i := 0; i < g.config.Users; i++ {
				dept, _ := getDepartment(i, g.config.Users)
				coverage, inTargetDept := app.DeptCoverage[dept]

				assign := false
				if inTargetDept {
					assign = shouldAssign(i, g.currentGroup, coverage)
				} else if app.NoisePct > 0 {
					// Cross-department noise access
					assign = shouldAssign(i, g.currentGroup+1000, app.NoisePct)
				}

				if assign {
					members = append(members, userId(i))
					// Make ~2% of members admins
					if shouldAssign(i, g.currentGroup, 2) {
						admins = append(admins, userId(i))
					}
				}
			}

			db := &dbResource{
				Group: &Group{
					Id:        groupId(g.currentGroup),
					Name:      app.Name,
					Admins:    admins,
					Members:   members,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
			g.currentGroup++
			return db, true
		}

		// Everyone group
		groupMembers := make([]string, 0, g.config.Users)
		for i := 0; i < g.config.Users; i++ {
			groupMembers = append(groupMembers, userId(i))
		}
		db := &dbResource{
			Group: &Group{
				Id:        "group-everyone",
				Name:      "Everyone",
				Admins:    []string{userId(0)},
				Members:   groupMembers,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.groupsDone = true
		return db, true
	}

	// Phase 2: Projects (use configured count)
	if g.currentProject < g.config.Projects {
		totalGroups := g.totalAppGroups()
		db := &dbResource{
			Project: &Project{
				Id:    fmt.Sprintf("project-%07d", g.currentProject),
				Name:  fmt.Sprintf("Project %07d", g.currentProject),
				Owner: userId(g.currentProject % g.config.Users),
				GroupAssignments: []string{
					groupId(g.currentProject % totalGroups),
					groupId((g.currentProject * 10) % totalGroups),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.currentProject++
		return db, true
	}

	// Phase 3: Roles (use configured count)
	if g.currentRole < g.config.Roles {
		totalGroups := g.totalAppGroups()
		directAssignments := []string{}
		if g.config.Users > 0 {
			directAssignments = append(directAssignments, userId(g.currentRole%g.config.Users))
			directAssignments = append(directAssignments, userId((g.currentRole*10)%g.config.Users))
		}
		groupAssignments := []string{}
		if totalGroups > 5 {
			groupAssignments = append(groupAssignments, groupId(g.currentRole%totalGroups))
			groupAssignments = append(groupAssignments, groupId((g.currentRole*10)%totalGroups))
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

	// Phase 4: Scoped roles (use configured count)
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

	// Phase 5: Users with department, job title, and manager
	if g.currentUser < g.config.Users {
		first, last := getUserName(g.currentUser)
		fullName := fmt.Sprintf("%s %s", first, last)
		email := fmt.Sprintf("%s.%s.%d@example.com", first, last, g.currentUser)
		dept, jobTitle := getDepartment(g.currentUser, g.config.Users)

		attrs := map[string]string{
			"full_name":  fullName,
			"email":      email,
			"department": dept,
			"job_title":  jobTitle,
		}

		if managerEmail := getManagerEmail(g.currentUser, g.config.Users); managerEmail != "" {
			attrs["manager_email"] = managerEmail
		}

		db := &dbResource{
			User: &User{
				Id:        userId(g.currentUser),
				Name:      fullName,
				Email:     email,
				Enabled:   true,
				Attrs:     attrs,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		g.currentUser++
		return db, true
	}

	// Phase 6: Passwords
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
