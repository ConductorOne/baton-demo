package client

import "time"

// DutyPrecision says how a named role's duties relate to the role.
type DutyPrecision string

const (
	// DutyPrecisionExact marks a role that exists to grant its one duty.
	DutyPrecisionExact DutyPrecision = "exact"
	// DutyPrecisionInclusive marks a broad role that allows its duties among
	// other things.
	DutyPrecisionInclusive DutyPrecision = "inclusive"
)

// NamedRole is a fixed, human-readable role created alongside the generated
// "Role 0000000"-style roles, like the fixed "Everyone" group. Named roles
// give separation-of-duties demos something meaningful to reason about.
type NamedRole struct {
	Id        string
	Name      string
	Duties    []string
	Precision DutyPrecision
}

// namedRoles is the single place that defines the named roles and the duties
// each one allows. Duty IDs are plain strings matching ConductorOne's duty
// catalog (<object>.<action>); they may change until the catalog names are
// final, so edit them here.
var namedRoles = []NamedRole{
	{Id: "role-payment-preparer", Name: "Payment Preparer", Precision: DutyPrecisionExact,
		Duties: []string{"payment.prepare"}},
	{Id: "role-payment-approver", Name: "Payment Approver", Precision: DutyPrecisionExact,
		Duties: []string{"payment.authorize"}},
	{Id: "role-payment-releaser", Name: "Payment Releaser", Precision: DutyPrecisionExact,
		Duties: []string{"payment.release"}},
	{Id: "role-finance-administrator", Name: "Finance Administrator", Precision: DutyPrecisionInclusive,
		Duties: []string{"payment.prepare", "payment.authorize", "payment.release", "vendor.activate"}},
	{Id: "role-developer", Name: "Developer", Precision: DutyPrecisionExact,
		Duties: []string{"software_change.author"}},
	{Id: "role-release-manager", Name: "Release Manager", Precision: DutyPrecisionInclusive,
		Duties: []string{"software_change.approve", "software_change.merge", "release.deploy"}},
}

// namedRoleSeedGrants deterministically assigns named roles to the first few
// users so separation-of-duties rules have conflicts to find. Users are
// referenced by generation index; an index beyond the configured user count is
// skipped.
var namedRoleSeedGrants = []struct {
	userIndex int
	roleIDs   []string
}{
	// Prepares and releases the same payment.
	{userIndex: 0, roleIDs: []string{"role-payment-preparer", "role-payment-releaser"}},
	// Authors a change and also approves, merges and deploys it.
	{userIndex: 1, roleIDs: []string{"role-developer", "role-release-manager"}},
	// One broad role that on its own allows every payment duty.
	{userIndex: 2, roleIDs: []string{"role-finance-administrator"}},
}

// NamedRoles returns the fixed named roles.
func NamedRoles() []NamedRole {
	out := make([]NamedRole, len(namedRoles))
	copy(out, namedRoles)
	return out
}

// NamedRoleByID returns the named role with the given ID, if there is one.
func NamedRoleByID(id string) (NamedRole, bool) {
	for _, r := range namedRoles {
		if r.Id == id {
			return r, true
		}
	}
	return NamedRole{}, false
}

// namedRole builds the database record for the i-th named role, with its
// seeded direct assignments.
func (g *generator) namedRole(i int) *Role {
	nr := namedRoles[i]
	directAssignments := []string{}
	for _, seed := range namedRoleSeedGrants {
		if seed.userIndex >= g.userCount() {
			continue
		}
		for _, roleID := range seed.roleIDs {
			if roleID == nr.Id {
				directAssignments = append(directAssignments, g.userID(seed.userIndex))
			}
		}
	}
	return &Role{
		Id:                nr.Id,
		Name:              nr.Name,
		DirectAssignments: directAssignments,
		GroupAssignments:  []string{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}
