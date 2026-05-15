package client

import (
	"fmt"
	"strings"
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
	App        *App
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
	case r.App != nil:
		return fmt.Sprintf("App: id %s name '%s' %d members %d child_groups", r.App.Id, r.App.Name, len(r.App.Members), len(r.App.ChildGroups))
	}
	return "Unknown"
}

// --- Organization hierarchy ---

const numCxO = 7
const numServiceAccounts = 4
const numSharedAccounts = 2
const numSpecialAccounts = numServiceAccounts + numSharedAccounts

type orgPosition struct {
	Title          string
	Department     string
	Level          string
	Region         string
	EmploymentType string // Full-time, Contractor, Service Account, Shared Account
	Enabled        bool
	ManagerIdx     int // -1 = no manager
}

type departmentDef struct {
	Name   string
	Weight int // cumulative percentage boundary (0-100)
	CxoIdx int // index into cSuitePositions
	Titles map[string][]string
}

type levelDef struct {
	Name   string
	Weight int // cumulative percentage boundary (0-100)
}

var cSuitePositions = [numCxO]struct{ Title string }{
	{"Chief Executive Officer"},
	{"Chief Technology Officer"},
	{"Chief Financial Officer"},
	{"Chief Operating Officer"},
	{"Chief Marketing Officer"},
	{"Chief Human Resources Officer"},
	{"Chief Revenue Officer"},
}

// departments defines the org. Weights are cumulative boundaries.
// Engineering=0-25, Sales=26-42, Customer Support=43-54, Marketing=55-64,
// Product=65-72, Finance=73-80, HR=81-87, Workplace & IT=88-92, Legal=93-99.
var departments = []departmentDef{
	{Name: "Engineering", Weight: 26, CxoIdx: 1, Titles: map[string][]string{
		"VP":         {"VP of Engineering"},
		"Director":   {"Director of Platform Engineering", "Director of Infrastructure", "Director of Security Engineering", "Director of QA"},
		"Manager":    {"Engineering Manager - Backend", "Engineering Manager - Frontend", "Engineering Manager - DevOps", "Engineering Manager - SRE", "Engineering Manager - Mobile", "Engineering Manager - Data"},
		"Senior IC":  {"Staff Engineer", "Senior Software Engineer", "Senior DevOps Engineer", "Senior Security Engineer", "Senior QA Engineer", "Senior Data Engineer"},
		"IC":         {"Software Engineer", "Frontend Engineer", "Backend Engineer", "DevOps Engineer", "QA Engineer", "Security Engineer", "Data Engineer"},
		"Contractor": {"Contract Software Engineer", "Contract QA Engineer"},
	}},
	{Name: "Sales", Weight: 43, CxoIdx: 6, Titles: map[string][]string{
		"VP":         {"VP of Sales"},
		"Director":   {"Director of Enterprise Sales", "Director of SMB Sales", "Director of Sales Operations"},
		"Manager":    {"Sales Manager - Enterprise", "Sales Manager - SMB", "Sales Manager - APAC", "Sales Manager - EMEA"},
		"Senior IC":  {"Senior Account Executive", "Senior Solutions Engineer", "Senior Sales Operations Analyst"},
		"IC":         {"Account Executive", "Sales Development Rep", "Solutions Engineer", "Sales Operations Analyst", "Business Development Rep"},
		"Contractor": {"Contract Sales Rep"},
	}},
	{Name: "Customer Support", Weight: 55, CxoIdx: 6, Titles: map[string][]string{
		"VP":         {"VP of Customer Support"},
		"Director":   {"Director of Support Operations", "Director of Technical Support"},
		"Manager":    {"CS Manager - Americas", "CS Manager - EMEA", "CS Manager - APAC", "CS Manager - Technical"},
		"Senior IC":  {"Senior Customer Success Manager", "Senior Support Engineer"},
		"IC":         {"Customer Support Rep", "Customer Success Manager", "Support Engineer", "Support Analyst"},
		"Contractor": {"Contract Support Rep"},
	}},
	{Name: "Marketing", Weight: 65, CxoIdx: 4, Titles: map[string][]string{
		"VP":         {"VP of Marketing"},
		"Director":   {"Director of Content & Brand", "Director of Growth Marketing", "Director of Events & Communications"},
		"Manager":    {"Marketing Manager - Content", "Marketing Manager - Digital", "Marketing Manager - Events"},
		"Senior IC":  {"Senior Content Strategist", "Senior Growth Analyst", "Senior Brand Designer"},
		"IC":         {"Content Writer", "SEO Specialist", "Growth Analyst", "Brand Designer", "Event Coordinator", "Social Media Manager"},
		"Contractor": {"Contract Designer", "Contract Copywriter"},
	}},
	{Name: "Product", Weight: 73, CxoIdx: 1, Titles: map[string][]string{
		"VP":         {"VP of Product"},
		"Director":   {"Director of Product Management", "Director of Design"},
		"Manager":    {"Lead Product Manager", "Design Manager"},
		"Senior IC":  {"Staff Product Manager", "Senior Product Designer", "Senior UX Researcher"},
		"IC":         {"Product Manager", "Product Designer", "UX Researcher", "Technical Writer", "Data Analyst"},
		"Contractor": {"Contract Product Designer"},
	}},
	{Name: "Finance", Weight: 81, CxoIdx: 2, Titles: map[string][]string{
		"VP":         {"VP of Finance"},
		"Director":   {"Director of Accounting", "Director of FP&A"},
		"Manager":    {"Finance Manager - AP/AR", "Finance Manager - Planning"},
		"Senior IC":  {"Senior Financial Analyst", "Senior Accountant"},
		"IC":         {"Financial Analyst", "Accountant", "FP&A Analyst", "Billing Specialist", "Accounts Payable Specialist"},
		"Contractor": {"Contract Accountant"},
	}},
	{Name: "Human Resources", Weight: 88, CxoIdx: 5, Titles: map[string][]string{
		"VP":         {"VP of Human Resources"},
		"Director":   {"Director of Talent Acquisition", "Director of HR Business Partners", "Director of Compensation & Benefits"},
		"Manager":    {"HR Manager", "Recruiting Manager", "Benefits Manager"},
		"Senior IC":  {"Senior HR Business Partner", "Senior Recruiter", "Senior Compensation Analyst"},
		"IC":         {"HR Business Partner", "Recruiter", "Compensation Analyst", "People Operations Coordinator", "HR Coordinator"},
		"Contractor": {"Contract Recruiter"},
	}},
	{Name: "Workplace & IT", Weight: 93, CxoIdx: 3, Titles: map[string][]string{
		"VP":         {"VP of Workplace & IT"},
		"Director":   {"Director of IT Operations", "Director of Facilities"},
		"Manager":    {"IT Manager", "Facilities Manager"},
		"Senior IC":  {"Senior Systems Administrator", "Senior IT Engineer"},
		"IC":         {"IT Support Specialist", "Systems Administrator", "Office Coordinator", "Facilities Coordinator", "Help Desk Analyst"},
		"Contractor": {"Contract IT Support"},
	}},
	{Name: "Legal", Weight: 100, CxoIdx: 2, Titles: map[string][]string{
		"VP":         {"VP & General Counsel"},
		"Director":   {"Director of Legal", "Director of Compliance"},
		"Manager":    {"Legal Operations Manager", "Compliance Manager"},
		"Senior IC":  {"Senior Corporate Counsel", "Senior Compliance Analyst"},
		"IC":         {"Paralegal", "Compliance Analyst", "Legal Coordinator", "Contract Administrator"},
		"Contractor": {"Contract Paralegal"},
	}},
}

// VP=0-3%, Director=4-12%, Manager=13-26%, Senior IC=27-51%, IC=52-91%, Contractor=92-99%.
var levelDistribution = []levelDef{
	{"VP", 4},
	{"Director", 13},
	{"Manager", 27},
	{"Senior IC", 52},
	{"IC", 92},
	{"Contractor", 100},
}

var regionDistribution = []struct {
	Name   string
	Weight int
}{
	{"Americas", 55},
	{"EMEA", 80},
	{"APAC", 100},
}

var levelOrder = map[string]int{
	"C-Suite": 6, "VP": 5, "Director": 4, "Manager": 3,
	"Senior IC": 2, "IC": 1, "Contractor": 0,
}

var parentLevel = map[string]string{
	"VP": "C-Suite", "Director": "VP", "Manager": "Director",
	"Senior IC": "Manager", "IC": "Manager", "Contractor": "Manager",
}

var serviceAccountDefs = [numServiceAccounts]struct {
	Name, Email, Dept string
}{
	{"CI/CD Pipeline Bot", "cicd-bot@service.example.com", "Engineering"},
	{"Monitoring Service", "monitoring-svc@service.example.com", "Engineering"},
	{"Integration Service", "integration-svc@service.example.com", "Workplace & IT"},
	{"Analytics Pipeline", "analytics-pipeline@service.example.com", "Product"},
}

var sharedAccountDefs = [numSharedAccounts]struct {
	Name, Email, Dept string
}{
	{"Sales Demo Environment", "sales-demo@shared.example.com", "Sales"},
	{"QA Test Environment", "qa-test@shared.example.com", "Engineering"},
}

// --- SaaS application groups ---

type appDef struct {
	Name         string
	DeptCoverage map[string]int
	NoisePct     int
	MinLevel     string // minimum seniority level (empty = any)
	RegionOnly   string // restrict to region (empty = all regions)
}

func allDepts(coverage int) map[string]int {
	return map[string]int{
		"Engineering": coverage, "Sales": coverage, "Marketing": coverage,
		"Finance": coverage, "Human Resources": coverage, "Product": coverage,
		"Customer Support": coverage, "Workplace & IT": coverage,
		"Legal": coverage, "Executive": coverage,
	}
}

var appGroups = []appDef{
	// ==================================================================
	// Google Workspace (parent + children)
	// ==================================================================
	{Name: "Google Workspace", DeptCoverage: allDepts(100)},
	{Name: "Google Workspace - Admin Console", DeptCoverage: map[string]int{"Workplace & IT": 80, "Engineering": 10}, NoisePct: 1},
	{Name: "Google Workspace - Drive Manager", DeptCoverage: map[string]int{
		"Workplace & IT": 70, "Legal": 40, "Human Resources": 35, "Finance": 30,
	}, NoisePct: 2},
	{Name: "Google Workspace - Vault", DeptCoverage: map[string]int{"Legal": 85, "Human Resources": 40, "Finance": 25}, NoisePct: 1},
	{Name: "Google Workspace - Groups Admin", DeptCoverage: map[string]int{"Workplace & IT": 65, "Human Resources": 20}, NoisePct: 1},

	// ==================================================================
	// Slack (parent + children)
	// ==================================================================
	{Name: "Slack", DeptCoverage: allDepts(100)},
	{Name: "Slack - Workspace Admin", DeptCoverage: map[string]int{"Workplace & IT": 70, "Engineering": 5}, NoisePct: 1},
	{Name: "Slack - Channel Management", DeptCoverage: map[string]int{"Workplace & IT": 50, "Human Resources": 20, "Marketing": 15}, NoisePct: 1},
	{Name: "Slack - App Installation", DeptCoverage: map[string]int{"Workplace & IT": 45, "Engineering": 10}, NoisePct: 1},
	{Name: "Slack - Analytics", DeptCoverage: map[string]int{"Workplace & IT": 55, "Human Resources": 15}, NoisePct: 1},

	// ==================================================================
	// Okta (parent + children)
	// ==================================================================
	{Name: "Okta", DeptCoverage: allDepts(100)},
	{Name: "Okta - Admin Console", DeptCoverage: map[string]int{"Workplace & IT": 85, "Engineering": 15}, NoisePct: 1},
	{Name: "Okta - App Management", DeptCoverage: map[string]int{"Workplace & IT": 70}, NoisePct: 1},
	{Name: "Okta - User Lifecycle", DeptCoverage: map[string]int{"Workplace & IT": 60, "Human Resources": 30}, NoisePct: 1},
	{Name: "Okta - MFA Policy Admin", DeptCoverage: map[string]int{"Workplace & IT": 55, "Engineering": 8}, NoisePct: 1},

	// ==================================================================
	// Other universal apps (no children)
	// ==================================================================
	{Name: "Zoom", DeptCoverage: allDepts(100)},
	{Name: "1Password", DeptCoverage: allDepts(97)},
	{Name: "Confluence", DeptCoverage: map[string]int{
		"Engineering": 95, "Sales": 70, "Marketing": 75, "Finance": 65,
		"Human Resources": 70, "Product": 93, "Customer Support": 60,
		"Workplace & IT": 80, "Legal": 55, "Executive": 85,
	}, NoisePct: 3},

	// ==================================================================
	// GitHub (parent + children)
	// ==================================================================
	{Name: "GitHub", DeptCoverage: map[string]int{"Engineering": 96, "Product": 35, "Executive": 15}, NoisePct: 3},
	{Name: "GitHub - Org Owner", DeptCoverage: map[string]int{"Engineering": 5}},
	{Name: "GitHub - Actions Admin", DeptCoverage: map[string]int{"Engineering": 25}, NoisePct: 1},
	{Name: "GitHub - Security Alerts", DeptCoverage: map[string]int{"Engineering": 30, "Workplace & IT": 20}, NoisePct: 1},
	{Name: "GitHub - Code Scanning", DeptCoverage: map[string]int{"Engineering": 20}, NoisePct: 1},
	{Name: "GitHub - Copilot", DeptCoverage: map[string]int{"Engineering": 70, "Product": 15}, NoisePct: 2},
	{Name: "GitHub - Enterprise Admin", DeptCoverage: map[string]int{"Engineering": 3, "Workplace & IT": 10}},

	// ==================================================================
	// Jira (parent + children)
	// ==================================================================
	{Name: "Jira", DeptCoverage: map[string]int{"Engineering": 93, "Product": 88, "Marketing": 25, "Customer Support": 30}, NoisePct: 4},
	{Name: "Jira - Admin", DeptCoverage: map[string]int{"Engineering": 12, "Product": 8, "Workplace & IT": 15}, NoisePct: 1},
	{Name: "Jira - Board Management", DeptCoverage: map[string]int{"Engineering": 40, "Product": 50}, NoisePct: 1},
	{Name: "Jira - Automation Rules", DeptCoverage: map[string]int{"Engineering": 18, "Product": 12}, NoisePct: 1},
	{Name: "Jira - Service Management", DeptCoverage: map[string]int{"Customer Support": 75, "Workplace & IT": 60, "Engineering": 20}, NoisePct: 2},

	// ==================================================================
	// AWS (parent + children)
	// ==================================================================
	{Name: "AWS Console", DeptCoverage: map[string]int{"Engineering": 88, "Executive": 10}, NoisePct: 2},
	{Name: "AWS - Production Deploy", DeptCoverage: map[string]int{"Engineering": 45}},
	{Name: "AWS - Production DB Admin", DeptCoverage: map[string]int{"Engineering": 35}},
	{Name: "AWS - S3 Storage", DeptCoverage: map[string]int{"Engineering": 60, "Product": 10}, NoisePct: 1},
	{Name: "AWS - IAM Admin", DeptCoverage: map[string]int{"Engineering": 10, "Workplace & IT": 30}},
	{Name: "AWS - CloudWatch", DeptCoverage: map[string]int{"Engineering": 55}, NoisePct: 1},
	{Name: "AWS - Cost Explorer", DeptCoverage: map[string]int{"Finance": 50, "Engineering": 15, "Executive": 20}, NoisePct: 1},
	{Name: "AWS - Lambda", DeptCoverage: map[string]int{"Engineering": 50}, NoisePct: 1},
	{Name: "AWS - EKS Admin", DeptCoverage: map[string]int{"Engineering": 25}},

	// ==================================================================
	// Datadog (parent + children)
	// ==================================================================
	{Name: "Datadog", DeptCoverage: map[string]int{"Engineering": 83}, NoisePct: 1},
	{Name: "Datadog - Admin", DeptCoverage: map[string]int{"Engineering": 12}, NoisePct: 1},
	{Name: "Datadog - Dashboard Management", DeptCoverage: map[string]int{"Engineering": 40, "Product": 10}, NoisePct: 1},
	{Name: "Datadog - Monitors & Alerts", DeptCoverage: map[string]int{"Engineering": 55}, NoisePct: 1},

	// Other engineering tools (no children)
	{Name: "PagerDuty", DeptCoverage: map[string]int{"Engineering": 74}, NoisePct: 1},
	{Name: "CircleCI", DeptCoverage: map[string]int{"Engineering": 68}, NoisePct: 1},
	{Name: "Docker Hub", DeptCoverage: map[string]int{"Engineering": 72}, NoisePct: 1},
	{Name: "Terraform Cloud", DeptCoverage: map[string]int{"Engineering": 55}, NoisePct: 1},

	// ==================================================================
	// Salesforce (parent + children)
	// ==================================================================
	{Name: "Salesforce", DeptCoverage: map[string]int{"Sales": 96, "Customer Support": 40, "Executive": 20}, NoisePct: 2},
	{Name: "Salesforce - Admin", DeptCoverage: map[string]int{"Sales": 8, "Workplace & IT": 15}},
	{Name: "Salesforce - Sales Cloud", DeptCoverage: map[string]int{"Sales": 85}, NoisePct: 1},
	{Name: "Salesforce - Service Cloud", DeptCoverage: map[string]int{"Customer Support": 80, "Sales": 25}, NoisePct: 1},
	{Name: "Salesforce - Reports & Dashboards", DeptCoverage: map[string]int{"Sales": 50, "Finance": 30, "Executive": 40}, NoisePct: 2},
	{Name: "Salesforce - CPQ", DeptCoverage: map[string]int{"Sales": 40}, NoisePct: 1},
	{Name: "Salesforce - Marketing Cloud", DeptCoverage: map[string]int{"Marketing": 60, "Sales": 15}, NoisePct: 1},

	// ==================================================================
	// HubSpot (parent + children)
	// ==================================================================
	{Name: "HubSpot", DeptCoverage: map[string]int{"Sales": 91, "Marketing": 88}, NoisePct: 3},
	{Name: "HubSpot - Admin", DeptCoverage: map[string]int{"Sales": 8, "Marketing": 10, "Workplace & IT": 12}},
	{Name: "HubSpot - Marketing Hub", DeptCoverage: map[string]int{"Marketing": 80}, NoisePct: 1},
	{Name: "HubSpot - Sales Hub", DeptCoverage: map[string]int{"Sales": 82}, NoisePct: 1},
	{Name: "HubSpot - CMS Hub", DeptCoverage: map[string]int{"Marketing": 45, "Product": 10}, NoisePct: 1},

	// Other sales tools (no children)
	{Name: "Gong", DeptCoverage: map[string]int{"Sales": 87}, NoisePct: 2},
	{Name: "Outreach", DeptCoverage: map[string]int{"Sales": 78}, NoisePct: 1},
	{Name: "LinkedIn Sales Navigator", DeptCoverage: map[string]int{"Sales": 73}, NoisePct: 1},
	{Name: "Clari", DeptCoverage: map[string]int{"Sales": 65}, NoisePct: 1},

	// ==================================================================
	// Zendesk (parent + children)
	// ==================================================================
	{Name: "Zendesk", DeptCoverage: map[string]int{"Customer Support": 95, "Sales": 20, "Product": 15}, NoisePct: 2},
	{Name: "Zendesk - Admin", DeptCoverage: map[string]int{"Customer Support": 10, "Workplace & IT": 15}},
	{Name: "Zendesk - Ticket Automation", DeptCoverage: map[string]int{"Customer Support": 45, "Engineering": 8}, NoisePct: 1},
	{Name: "Zendesk - Knowledge Base", DeptCoverage: map[string]int{"Customer Support": 60, "Product": 20, "Marketing": 10}, NoisePct: 2},
	{Name: "Zendesk - Explore Analytics", DeptCoverage: map[string]int{"Customer Support": 35, "Product": 15}, NoisePct: 1},

	// Other CS tools
	{Name: "Intercom", DeptCoverage: map[string]int{"Customer Support": 88, "Product": 25, "Marketing": 15}, NoisePct: 2},

	// ==================================================================
	// Figma (parent + children)
	// ==================================================================
	{Name: "Figma", DeptCoverage: map[string]int{"Marketing": 91, "Product": 89, "Engineering": 22}, NoisePct: 3},
	{Name: "Figma - Admin", DeptCoverage: map[string]int{"Product": 8, "Marketing": 5, "Workplace & IT": 10}},
	{Name: "Figma - Organization Libraries", DeptCoverage: map[string]int{"Product": 60, "Marketing": 55, "Engineering": 10}, NoisePct: 1},
	{Name: "Figma - Dev Mode", DeptCoverage: map[string]int{"Engineering": 18, "Product": 30}, NoisePct: 1},

	// Other marketing tools (no children)
	{Name: "Google Analytics", DeptCoverage: map[string]int{"Marketing": 90, "Product": 82}, NoisePct: 3},
	{Name: "Canva", DeptCoverage: map[string]int{"Marketing": 83}, NoisePct: 2},
	{Name: "Mailchimp", DeptCoverage: map[string]int{"Marketing": 78}, NoisePct: 1},
	{Name: "Marketo", DeptCoverage: map[string]int{"Marketing": 72}, NoisePct: 1},
	{Name: "Hootsuite", DeptCoverage: map[string]int{"Marketing": 65}, NoisePct: 1},

	// ---- Product tools (no children) ----
	{Name: "Amplitude", DeptCoverage: map[string]int{"Product": 89, "Engineering": 30}, NoisePct: 2},
	{Name: "FullStory", DeptCoverage: map[string]int{"Product": 78, "Customer Support": 20}, NoisePct: 1},
	{Name: "Notion", DeptCoverage: map[string]int{"Product": 87, "Engineering": 72, "Marketing": 40}, NoisePct: 4},
	{Name: "Miro", DeptCoverage: map[string]int{"Product": 80, "Engineering": 45, "Marketing": 35}, NoisePct: 3},
	{Name: "Productboard", DeptCoverage: map[string]int{"Product": 75, "Customer Support": 15}, NoisePct: 1},

	// ==================================================================
	// NetSuite (parent + children)
	// ==================================================================
	{Name: "NetSuite", DeptCoverage: map[string]int{"Finance": 96}, NoisePct: 1},
	{Name: "NetSuite - Admin", DeptCoverage: map[string]int{"Finance": 15}},
	{Name: "NetSuite - Financial Reports", DeptCoverage: map[string]int{"Finance": 80, "Executive": 30}, NoisePct: 1},
	{Name: "NetSuite - Payment Creation", DeptCoverage: map[string]int{"Finance": 70}},
	{Name: "NetSuite - Payment Approval", DeptCoverage: map[string]int{"Finance": 65}},
	{Name: "NetSuite - Inventory Management", DeptCoverage: map[string]int{"Finance": 40}, NoisePct: 1},
	{Name: "NetSuite - Vendor Portal", DeptCoverage: map[string]int{"Finance": 50}, NoisePct: 1},

	// Other finance tools (no children)
	{Name: "Expensify", DeptCoverage: map[string]int{"Finance": 92}, NoisePct: 2},
	{Name: "Stripe Dashboard", DeptCoverage: map[string]int{"Finance": 76, "Engineering": 28}, NoisePct: 2},
	{Name: "QuickBooks", DeptCoverage: map[string]int{"Finance": 69}, NoisePct: 1},
	{Name: "Coupa", DeptCoverage: map[string]int{"Finance": 60}, NoisePct: 1},

	// ==================================================================
	// Workday (parent + children)
	// ==================================================================
	{Name: "Workday", DeptCoverage: map[string]int{"Human Resources": 97}, NoisePct: 1},
	{Name: "Workday - Admin", DeptCoverage: map[string]int{"Human Resources": 40}},
	{Name: "Workday - Compensation Admin", DeptCoverage: map[string]int{"Human Resources": 30, "Finance": 15}},
	{Name: "Workday - Benefits Admin", DeptCoverage: map[string]int{"Human Resources": 35}},
	{Name: "Workday - Reporting", DeptCoverage: map[string]int{"Human Resources": 60, "Executive": 20, "Finance": 10}, NoisePct: 1},
	{Name: "Workday - Recruiting", DeptCoverage: map[string]int{"Human Resources": 70}, NoisePct: 1},
	{Name: "Workday - Learning", DeptCoverage: map[string]int{"Human Resources": 55}, NoisePct: 1},

	// Other HR tools (no children)
	{Name: "Greenhouse", DeptCoverage: map[string]int{"Human Resources": 93}, NoisePct: 2},
	{Name: "BambooHR", DeptCoverage: map[string]int{"Human Resources": 88}, NoisePct: 1},
	{Name: "Culture Amp", DeptCoverage: map[string]int{"Human Resources": 73}, NoisePct: 1},
	{Name: "Lattice", DeptCoverage: map[string]int{"Human Resources": 70}, NoisePct: 1},

	// ==================================================================
	// ServiceNow (parent + children)
	// ==================================================================
	{Name: "ServiceNow", DeptCoverage: map[string]int{"Workplace & IT": 92, "Engineering": 15}, NoisePct: 2},
	{Name: "ServiceNow - Admin", DeptCoverage: map[string]int{"Workplace & IT": 40}},
	{Name: "ServiceNow - Incident Management", DeptCoverage: map[string]int{"Workplace & IT": 75, "Engineering": 30}, NoisePct: 1},
	{Name: "ServiceNow - Change Management", DeptCoverage: map[string]int{"Workplace & IT": 65, "Engineering": 20}, NoisePct: 1},
	{Name: "ServiceNow - CMDB", DeptCoverage: map[string]int{"Workplace & IT": 55, "Engineering": 15}, NoisePct: 1},
	{Name: "ServiceNow - Asset Management", DeptCoverage: map[string]int{"Workplace & IT": 50, "Finance": 15}, NoisePct: 1},

	// Other Workplace & IT tools (no children)
	{Name: "Jamf", DeptCoverage: map[string]int{"Workplace & IT": 85}, NoisePct: 1},
	{Name: "Duo Security", DeptCoverage: map[string]int{"Workplace & IT": 90, "Engineering": 50}, NoisePct: 2},

	// ==================================================================
	// DocuSign (parent + children)
	// ==================================================================
	{Name: "DocuSign", DeptCoverage: map[string]int{"Legal": 95, "Sales": 25, "Finance": 30, "Human Resources": 20}, NoisePct: 3},
	{Name: "DocuSign - Admin", DeptCoverage: map[string]int{"Legal": 20, "Workplace & IT": 15}},
	{Name: "DocuSign - Template Management", DeptCoverage: map[string]int{"Legal": 60, "Sales": 15, "Human Resources": 10}, NoisePct: 1},
	{Name: "DocuSign - PowerForms", DeptCoverage: map[string]int{"Legal": 40, "Human Resources": 15}, NoisePct: 1},

	// Other legal tools
	{Name: "Ironclad", DeptCoverage: map[string]int{"Legal": 88}, NoisePct: 1},

	// ==================================================================
	// Management tools (Manager+)
	// ==================================================================
	{Name: "15Five", DeptCoverage: allDepts(95), MinLevel: "Manager"},
	{Name: "Workday Manager Portal", DeptCoverage: allDepts(93), MinLevel: "Manager"},
	{Name: "Lever Hiring Manager", DeptCoverage: map[string]int{
		"Engineering": 80, "Sales": 70, "Marketing": 65, "Finance": 60,
		"Human Resources": 95, "Product": 75, "Customer Support": 65,
		"Workplace & IT": 60, "Legal": 55, "Executive": 90,
	}, MinLevel: "Manager"},

	// ==================================================================
	// Leadership tools (Director+)
	// ==================================================================
	{Name: "Adaptive Insights", DeptCoverage: allDepts(90), MinLevel: "Director"},
	{Name: "Headcount Planning", DeptCoverage: allDepts(88), MinLevel: "Director"},

	// ==================================================================
	// Executive tools (VP+)
	// ==================================================================
	{Name: "Board Report Portal", DeptCoverage: allDepts(100), MinLevel: "VP"},
	{Name: "Strategic Planning Suite", DeptCoverage: allDepts(100), MinLevel: "VP"},
	{Name: "Executive Dashboard", DeptCoverage: map[string]int{"Executive": 100}, MinLevel: "C-Suite"},

	// ==================================================================
	// Regional compliance tools
	// ==================================================================
	{Name: "GDPR Compliance Portal", DeptCoverage: map[string]int{
		"Legal": 95, "Engineering": 40, "Human Resources": 50, "Workplace & IT": 60,
	}, RegionOnly: "EMEA", NoisePct: 5},
	{Name: "SOC2 Dashboard", DeptCoverage: map[string]int{
		"Legal": 90, "Engineering": 35, "Workplace & IT": 55,
	}, RegionOnly: "Americas", NoisePct: 3},
	{Name: "APAC CRM Module", DeptCoverage: map[string]int{
		"Sales": 90, "Customer Support": 85,
	}, RegionOnly: "APAC", NoisePct: 2},
}

// --- App-to-group mapping ---

type appMapping struct {
	ParentIdx    int
	ChildIndices []int
}

// buildAppMappings walks appGroups and identifies which entries are top-level
// apps vs child groups. A child group has a name like "ParentApp - ChildName"
// where ParentApp matches a preceding entry that has no " - " in its name.
func buildAppMappings() []appMapping {
	// First, collect all top-level (parent) names and their indices
	type parentInfo struct {
		idx  int
		name string
	}
	var parents []parentInfo
	isChild := make(map[int]bool)

	for i, ag := range appGroups {
		if !strings.Contains(ag.Name, " - ") {
			parents = append(parents, parentInfo{idx: i, name: ag.Name})
		}
	}

	// For each entry with " - ", check if its prefix matches a known parent
	for i, ag := range appGroups {
		if idx := strings.Index(ag.Name, " - "); idx > 0 {
			prefix := ag.Name[:idx]
			for _, p := range parents {
				if p.name == prefix {
					isChild[i] = true
					break
				}
			}
		}
	}

	// Build mappings
	var mappings []appMapping
	for _, p := range parents {
		m := appMapping{ParentIdx: p.idx}
		for i, ag := range appGroups {
			if isChild[i] {
				prefix := ag.Name[:strings.Index(ag.Name, " - ")]
				if prefix == p.name {
					m.ChildIndices = append(m.ChildIndices, i)
				}
			}
		}
		mappings = append(mappings, m)
	}

	return mappings
}

func appId(i int) string {
	return fmt.Sprintf("app-%07d", i)
}

// --- Multinational names ---

var firstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda",
	"David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Christopher", "Karen", "Charles", "Lisa", "Daniel", "Nancy",
	"Matthew", "Ashley", "Anthony", "Emily", "Mark", "Donna", "Steven", "Kimberly",
	"Carlos", "Maria", "Luis", "Isabella", "Diego", "Sofia", "Miguel", "Valentina",
	"Alejandro", "Camila", "Fernando", "Lucia", "Wei", "Yuki", "Hiroshi", "Mei",
	"Jun", "Sakura", "Kenji", "Ling", "Takeshi", "Hana", "Min", "Soo",
	"Raj", "Priya", "Arun", "Deepa", "Vikram", "Anita", "Sanjay", "Kavitha",
	"Arjun", "Neha", "Ravi", "Sunita", "Erik", "Astrid", "Pierre", "Amelie",
	"Hans", "Ingrid", "Marco", "Giulia", "Stefan", "Katarina", "Liam", "Saoirse",
	"Kwame", "Amara", "Olumide", "Fatima", "Tendai", "Zainab", "Omar", "Layla",
	"Hassan", "Noor", "Ali", "Yasmin", "Yusuf", "Aisha", "Pavel", "Elena",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Davis", "Miller", "Wilson",
	"Taylor", "Anderson", "Thomas", "Moore", "Jackson", "Martin", "Thompson", "White",
	"Harris", "Clark", "Lewis", "Robinson", "Walker", "Young", "Allen", "King",
	"Garcia", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Sanchez",
	"Ramirez", "Torres", "Flores", "Rivera", "Cruz", "Reyes", "Morales", "Gutierrez",
	"Chen", "Wang", "Li", "Zhang", "Liu", "Patel", "Sharma", "Kumar",
	"Singh", "Kim", "Park", "Nguyen", "Tanaka", "Yamamoto", "Sato",
	"Mueller", "Schmidt", "Johansson", "Eriksson", "Dubois", "Laurent", "Rossi",
	"Bianchi", "O'Brien", "O'Connor", "Kowalski", "Okafor", "Mensah", "Diallo",
	"Ibrahim", "Al-Rashid", "Nasser", "Abboud", "Petrov", "Volkov", "Santos",
	"Ferreira", "Nakamura", "Takahashi", "Andersen", "Larsen", "Virtanen", "Makinen",
	"Osei", "Adeyemi", "Kone", "Ben-David", "Cohen", "Novak", "Horvat",
}

// --- Edge case detection ---

// ~2.5% of IC/Senior IC/Contractor are disabled former employees with stale group access.
func isDisabledAccount(userIdx, totalUsers int, level string) bool {
	if totalUsers < 50 || userIdx < numCxO {
		return false
	}
	if level != "IC" && level != "Senior IC" && level != "Contractor" {
		return false
	}
	return userIdx%41 == 40
}

// ~1.5% have no manager (orphaned in the hierarchy).
func isOrphaned(userIdx, totalUsers int) bool {
	if totalUsers < 50 {
		return false
	}
	return userIdx > 30 && userIdx%47 == 46
}

// A few ICs/Senior ICs have access far beyond their level.
func isOverPrivileged(userIdx, totalUsers int, level string) bool {
	if totalUsers < 50 {
		return false
	}
	if level != "IC" && level != "Senior IC" {
		return false
	}
	return userIdx%53 == 52
}

// Some Product/CS employees transferred from another department and retain old access.
func isTransferred(userIdx, totalUsers int, dept string) bool {
	if totalUsers < 50 {
		return false
	}
	if dept != "Product" && dept != "Customer Support" {
		return false
	}
	return userIdx%43 == 42
}

func getTransferredFromDept(userIdx int, dept string) string {
	if dept == "Product" {
		return "Engineering"
	}
	return "Sales"
}

// A few Senior ICs act as interim managers and have manager-level tool access.
func isActingManager(userIdx, totalUsers int, level string) bool {
	if totalUsers < 50 {
		return false
	}
	if level != "Senior IC" {
		return false
	}
	return userIdx%59 == 58
}

// --- Org chart construction ---

func buildOrgChart(totalUsers int) []orgPosition {
	positions := make([]orgPosition, totalUsers)

	effectiveCxO := numCxO
	if totalUsers < numCxO {
		effectiveCxO = totalUsers
	}
	effectiveSpecial := numSpecialAccounts
	if totalUsers <= numCxO+numSpecialAccounts {
		effectiveSpecial = 0
	}

	// C-Suite (always Americas HQ)
	for i := 0; i < effectiveCxO; i++ {
		positions[i] = orgPosition{
			Title:          cSuitePositions[i].Title,
			Department:     "Executive",
			Level:          "C-Suite",
			Region:         "Americas",
			EmploymentType: "Full-time",
			Enabled:        true,
			ManagerIdx:     -1,
		}
		if i > 0 {
			positions[i].ManagerIdx = 0
		}
	}
	if totalUsers <= effectiveCxO {
		return positions
	}

	// Regular employees
	regularStart := effectiveCxO
	regularEnd := totalUsers - effectiveSpecial
	regularCount := regularEnd - regularStart
	if regularCount <= 0 {
		return positions
	}

	for i := regularStart; i < regularEnd; i++ {
		relIdx := i - regularStart
		pct := (relIdx * 100) / regularCount

		var dept *departmentDef
		for j := range departments {
			if pct < departments[j].Weight {
				dept = &departments[j]
				break
			}
		}
		if dept == nil {
			dept = &departments[len(departments)-1]
		}

		deptStart, deptEnd := getDeptBounds(regularStart, regularCount, dept.Name)
		deptSize := deptEnd - deptStart
		if deptSize <= 0 {
			deptSize = 1
		}
		posInDept := i - deptStart

		levelPct := (posInDept * 100) / deptSize
		level := "IC"
		for _, l := range levelDistribution {
			if levelPct < l.Weight {
				level = l.Name
				break
			}
		}

		titles := dept.Titles[level]
		title := level
		if len(titles) > 0 {
			title = titles[posInDept%len(titles)]
		}

		empType := "Full-time"
		if level == "Contractor" {
			empType = "Contractor"
		}

		enabled := !isDisabledAccount(i, totalUsers, level)

		positions[i] = orgPosition{
			Title:          title,
			Department:     dept.Name,
			Level:          level,
			Region:         getRegionForUser(i),
			EmploymentType: empType,
			Enabled:        enabled,
			ManagerIdx:     -1,
		}
	}

	// Special accounts
	specialStart := regularEnd
	for j := 0; j < numServiceAccounts && specialStart+j < totalUsers; j++ {
		sa := serviceAccountDefs[j]
		positions[specialStart+j] = orgPosition{
			Title:          sa.Name,
			Department:     sa.Dept,
			Level:          "IC",
			Region:         "Americas",
			EmploymentType: "Service Account",
			Enabled:        true,
			ManagerIdx:     -1,
		}
	}
	for j := 0; j < numSharedAccounts && specialStart+numServiceAccounts+j < totalUsers; j++ {
		sa := sharedAccountDefs[j]
		positions[specialStart+numServiceAccounts+j] = orgPosition{
			Title:          sa.Name,
			Department:     sa.Dept,
			Level:          "IC",
			Region:         "Americas",
			EmploymentType: "Shared Account",
			Enabled:        true,
			ManagerIdx:     -1,
		}
	}

	// Manager chain (pass 2)
	for i := regularStart; i < regularEnd; i++ {
		if isOrphaned(i, totalUsers) {
			continue
		}
		positions[i].ManagerIdx = findManager(positions, i, regularStart, regularEnd)
	}

	return positions
}

func getDeptBounds(regularStart, regularCount int, deptName string) (int, int) {
	prevWeight := 0
	for _, d := range departments {
		start := regularStart + (prevWeight * regularCount / 100)
		end := regularStart + (d.Weight * regularCount / 100)
		if d.Name == deptName {
			return start, end
		}
		prevWeight = d.Weight
	}
	return regularStart, regularStart
}

func getRegionForUser(userIdx int) string {
	h := uint32(userIdx) * 2654435761
	h ^= h >> 16
	pct := int(h % 100)
	for _, r := range regionDistribution {
		if pct < r.Weight {
			return r.Name
		}
	}
	return "Americas"
}

func findManager(positions []orgPosition, userIdx, regularStart, regularEnd int) int {
	pos := positions[userIdx]
	targetLevel, ok := parentLevel[pos.Level]
	if !ok {
		return -1
	}

	dept := pos.Department

	for targetLevel != "" {
		if targetLevel == "C-Suite" {
			for _, d := range departments {
				if d.Name == dept {
					return d.CxoIdx
				}
			}
			return 0
		}

		var candidates []int
		for j := regularStart; j < regularEnd; j++ {
			if j == userIdx {
				continue
			}
			if positions[j].Department == dept && positions[j].Level == targetLevel && positions[j].Enabled {
				candidates = append(candidates, j)
			}
		}
		if len(candidates) > 0 {
			return candidates[userIdx%len(candidates)]
		}

		next, ok := parentLevel[targetLevel]
		if !ok {
			break
		}
		targetLevel = next
	}

	for _, d := range departments {
		if d.Name == dept {
			return d.CxoIdx
		}
	}
	return 0
}

// --- Data generation helpers ---

func getUserName(userIdx int) (string, string) {
	first := firstNames[userIdx%len(firstNames)]
	last := lastNames[(userIdx*7+userIdx/len(firstNames))%len(lastNames)]
	return first, last
}

// shouldAssign is a deterministic hash-like function for group membership decisions.
func shouldAssign(userIdx, groupIdx, coveragePct int) bool {
	if coveragePct >= 100 {
		return true
	}
	if coveragePct <= 0 {
		return false
	}
	h := uint32(userIdx)*2654435761 + uint32(groupIdx)*2246822519
	h ^= h >> 16
	h *= 0x45d9f3b
	h ^= h >> 16
	return int(h%100) < coveragePct
}

func userId(i int) string {
	return fmt.Sprintf("user-%07d", i)
}

func groupId(i int) string {
	return fmt.Sprintf("group-%07d", i)
}

// --- Generator ---

type generator struct {
	config            *config.Demo
	positions         []orgPosition
	currentUser       int
	currentPassword   int
	currentGroup      int
	currentRole       int
	currentScopedRole int
	currentProject    int
	currentApp        int
	groupsDone        bool
	appsDone          bool
	appMappings       []appMapping
}

func (g *generator) ensurePositions() {
	if g.positions != nil {
		return
	}
	g.positions = buildOrgChart(g.config.Users)
	g.appMappings = buildAppMappings()
}

func (g *generator) totalAppGroups() int {
	return len(appGroups)
}

// validGroupIndices returns the appGroups indices that are actual groups (not top-level apps).
func (g *generator) validGroupIndices() []int {
	var indices []int
	for i := range appGroups {
		if !g.isTopLevelApp(i) {
			indices = append(indices, i)
		}
	}
	return indices
}

// isTopLevelApp returns true if the given appGroups index is a parent app entry.
func (g *generator) isTopLevelApp(groupIdx int) bool {
	for _, m := range g.appMappings {
		if m.ParentIdx == groupIdx {
			return true
		}
	}
	return false
}

// computeGroupMembers computes the members and admins for a given appGroups index.
func (g *generator) computeGroupMembers(groupIdx int) (members []string, admins []string) {
	app := appGroups[groupIdx]

	for i := 0; i < g.config.Users; i++ {
		pos := g.positions[i]

		if pos.EmploymentType == "Service Account" || pos.EmploymentType == "Shared Account" {
			if !g.isUniversalApp(groupIdx) {
				coverage, inDept := app.DeptCoverage[pos.Department]
				if !inDept || !shouldAssign(i, groupIdx, coverage) {
					continue
				}
			}
		}

		dept := pos.Department
		coverage, inTargetDept := app.DeptCoverage[dept]

		assign := false
		if inTargetDept {
			assign = shouldAssign(i, groupIdx, coverage)
		} else if app.NoisePct > 0 {
			assign = shouldAssign(i, groupIdx+1000, app.NoisePct)
		}

		// Level gate
		if app.MinLevel != "" && assign {
			userLvl := levelOrder[pos.Level]
			minLvl := levelOrder[app.MinLevel]
			if userLvl < minLvl {
				if app.MinLevel == "Manager" && isActingManager(i, g.config.Users, pos.Level) {
					// allowed
				} else {
					assign = false
				}
			}
		}

		// Region gate
		if app.RegionOnly != "" && assign {
			if pos.Region != app.RegionOnly {
				assign = false
			}
		}

		// Edge: transferred users retain old department access
		if !assign && isTransferred(i, g.config.Users, pos.Department) {
			oldDept := getTransferredFromDept(i, pos.Department)
			if oldCov, ok := app.DeptCoverage[oldDept]; ok {
				assign = shouldAssign(i, groupIdx, oldCov)
			}
		}

		// Edge: over-privileged ICs gain random extra access
		if !assign && isOverPrivileged(i, g.config.Users, pos.Level) {
			assign = shouldAssign(i, groupIdx+3000, 30)
		}

		if assign {
			members = append(members, userId(i))

			isAdmin := shouldAssign(i, groupIdx, 2)
			if isOverPrivileged(i, g.config.Users, pos.Level) {
				isAdmin = shouldAssign(i, groupIdx, 25)
			}
			if inTargetDept && levelOrder[pos.Level] >= levelOrder["VP"] {
				isAdmin = true
			}
			if isAdmin {
				admins = append(admins, userId(i))
			}
		}
	}

	return members, admins
}

func (g *generator) Next() (*dbResource, bool) {
	g.ensurePositions()

	// Phase 1: App-based groups (skip top-level app entries; they become app resources)
	if !g.groupsDone {
		for g.currentGroup < g.totalAppGroups() && g.isTopLevelApp(g.currentGroup) {
			g.currentGroup++
		}
		if g.currentGroup < g.totalAppGroups() {
			app := appGroups[g.currentGroup]
			members, admins := g.computeGroupMembers(g.currentGroup)

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

	// Phase 1.5: App resources
	if !g.appsDone {
		if g.currentApp < len(g.appMappings) {
			m := g.appMappings[g.currentApp]
			app := appGroups[m.ParentIdx]

			// App members = what would have been the top-level group's members
			members, _ := g.computeGroupMembers(m.ParentIdx)

			// child_groups = group IDs of the child entries
			var childGroups []string
			for _, childIdx := range m.ChildIndices {
				childGroups = append(childGroups, groupId(childIdx))
			}

			db := &dbResource{
				App: &App{
					Id:          appId(g.currentApp),
					Name:        app.Name,
					Members:     members,
					ChildGroups: childGroups,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				},
			}
			g.currentApp++
			return db, true
		}
		g.appsDone = true
	}

	// Phase 2: Projects
	if g.currentProject < g.config.Projects {
		validGroups := g.validGroupIndices()
		totalValid := len(validGroups)
		var groupAssignments []string
		if totalValid > 0 {
			groupAssignments = []string{
				groupId(validGroups[g.currentProject%totalValid]),
				groupId(validGroups[(g.currentProject*10)%totalValid]),
			}
		}
		db := &dbResource{
			Project: &Project{
				Id:               fmt.Sprintf("project-%07d", g.currentProject),
				Name:             fmt.Sprintf("Project %07d", g.currentProject),
				Owner:            userId(g.currentProject % g.config.Users),
				GroupAssignments: groupAssignments,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
		}
		g.currentProject++
		return db, true
	}

	// Phase 3: Roles
	if g.currentRole < g.config.Roles {
		validGroups := g.validGroupIndices()
		totalValid := len(validGroups)
		var directAssignments []string
		if g.config.Users > 0 {
			directAssignments = append(directAssignments, userId(g.currentRole%g.config.Users))
			directAssignments = append(directAssignments, userId((g.currentRole*10)%g.config.Users))
		}
		var groupAssignments []string
		if totalValid > 5 {
			groupAssignments = append(groupAssignments, groupId(validGroups[g.currentRole%totalValid]))
			groupAssignments = append(groupAssignments, groupId(validGroups[(g.currentRole*10)%totalValid]))
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

	// Phase 4: Scoped roles
	if g.currentScopedRole < g.config.ScopedRoles {
		var userAssignments []string
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

	// Phase 5: Users
	if g.currentUser < g.config.Users {
		pos := g.positions[g.currentUser]
		first, last := getUserName(g.currentUser)

		var fullName, email string
		switch pos.EmploymentType {
		case "Service Account":
			sa := serviceAccountDefs[g.currentUser-(g.config.Users-numSpecialAccounts)]
			fullName = sa.Name
			email = sa.Email
		case "Shared Account":
			sa := sharedAccountDefs[g.currentUser-(g.config.Users-numSharedAccounts)]
			fullName = sa.Name
			email = sa.Email
		default:
			fullName = fmt.Sprintf("%s %s", first, last)
			email = fmt.Sprintf("%s.%s.%d@example.com", first, last, g.currentUser)
		}

		attrs := map[string]string{
			"full_name":       fullName,
			"email":           email,
			"department":      pos.Department,
			"job_title":       pos.Title,
			"level":           pos.Level,
			"region":          pos.Region,
			"employment_type": pos.EmploymentType,
		}

		if pos.ManagerIdx >= 0 {
			mgrPos := g.positions[pos.ManagerIdx]
			var mgrEmail string
			switch mgrPos.EmploymentType {
			case "Service Account", "Shared Account":
				mgrEmail = ""
			default:
				mgrFirst, mgrLast := getUserName(pos.ManagerIdx)
				mgrEmail = fmt.Sprintf("%s.%s.%d@example.com", mgrFirst, mgrLast, pos.ManagerIdx)
			}
			if mgrEmail != "" {
				attrs["manager_email"] = mgrEmail
			}
		}

		db := &dbResource{
			User: &User{
				Id:        userId(g.currentUser),
				Name:      fullName,
				Email:     email,
				Enabled:   pos.Enabled,
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

// isUniversalApp returns true if every department has >=100% coverage.
func (g *generator) isUniversalApp(groupIdx int) bool {
	app := appGroups[groupIdx]
	for _, d := range departments {
		if app.DeptCoverage[d.Name] < 100 {
			return false
		}
	}
	return app.DeptCoverage["Executive"] >= 100
}

// --- Table descriptors (unchanged) ---

var allTableDescriptors = []tableDescriptor{
	users,
	groups,
	roles,
	scopedRoles,
	projects,
	passwords,
	apps,
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

var apps = (*appsTable)(nil)

type appsTable struct{}

func (t *appsTable) Name() string {
	return "apps"
}

func (t *appsTable) Schema() ([]string, []any) {
	return []string{
		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			members TEXT NOT NULL,
			child_groups TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}, []any{}
}
