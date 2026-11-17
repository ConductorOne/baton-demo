package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/duty/dutytest"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type expectedDuty struct {
	id        string
	precision v2.DutyMappingPrecision
}

var (
	exact     = v2.DutyMappingPrecision_DUTY_MAPPING_PRECISION_EXACT
	inclusive = v2.DutyMappingPrecision_DUTY_MAPPING_PRECISION_INCLUSIVE
)

// expectedNamedRoleDuties restates the named-role table independently so a
// change to it is a deliberate, reviewed change to this test as well.
var expectedNamedRoleDuties = map[string][]expectedDuty{
	"role-payment-preparer":      {{"payment.prepare", exact}},
	"role-payment-approver":      {{"payment.authorize", exact}},
	"role-payment-releaser":      {{"payment.release", exact}},
	"role-finance-administrator": {{"payment.prepare", inclusive}, {"payment.authorize", inclusive}, {"payment.release", inclusive}, {"vendor.activate", inclusive}},
	"role-developer":             {{"software_change.author", exact}},
	"role-release-manager":       {{"software_change.approve", inclusive}, {"software_change.merge", inclusive}, {"release.deploy", inclusive}},
}

// entitlementsAll drains every page of a builder's Entitlements for r.
func entitlementsAll(t *testing.T, b connectorbuilder.ResourceSyncerV2, r *v2.Resource) []*v2.Entitlement {
	t.Helper()
	var out []*v2.Entitlement
	token := ""
	for {
		ents, results, err := b.Entitlements(context.Background(), r, resource.SyncOpAttrs{
			PageToken: pagination.Token{Token: token},
		})
		require.NoError(t, err)
		out = append(out, ents...)
		if results == nil || results.NextPageToken == "" {
			break
		}
		token = results.NextPageToken
	}
	return out
}

// grantsAll drains every page of a builder's Grants for r.
func grantsAll(t *testing.T, b connectorbuilder.ResourceSyncerV2, r *v2.Resource) []*v2.Grant {
	t.Helper()
	var out []*v2.Grant
	token := ""
	for {
		grants, results, err := b.Grants(context.Background(), r, resource.SyncOpAttrs{
			PageToken: pagination.Token{Token: token},
		})
		require.NoError(t, err)
		out = append(out, grants...)
		if results == nil || results.NextPageToken == "" {
			break
		}
		token = results.NextPageToken
	}
	return out
}

func dutyMappings(t *testing.T, ent *v2.Entitlement) []*v2.DutyMapping {
	t.Helper()
	var out []*v2.DutyMapping
	for _, a := range ent.GetAnnotations() {
		if !a.MessageIs(&v2.DutyMappingsAnnotation{}) {
			continue
		}
		ann := &v2.DutyMappingsAnnotation{}
		require.NoError(t, a.UnmarshalTo(ann))
		out = append(out, ann.GetMappings()...)
	}
	return out
}

func roleResourceByID(t *testing.T, cl *client.Client, id string) *v2.Resource {
	t.Helper()
	for _, r := range listAll(t, newRoleBuilder(cl), 100) {
		if r.GetId().GetResource() == id {
			return r
		}
	}
	t.Fatalf("role %s not found", id)
	return nil
}

func TestEveryEntitlementPassesDutyAssertions(t *testing.T) {
	cl := testClient(t, estateConfig(t))
	demo := &Demo{client: cl}

	withDuties := map[string]int{}
	for _, b := range demo.ResourceSyncers(context.Background()) {
		for _, r := range listAll(t, b, 100) {
			for _, ent := range entitlementsAll(t, b, r) {
				dutytest.AssertEntitlement(t, ent)
				if n := len(dutyMappings(t, ent)); n > 0 {
					withDuties[ent.GetId()] = n
				}
			}
		}
	}

	// Only the named roles' assignment entitlements carry duties.
	want := map[string]int{}
	for roleID, duties := range expectedNamedRoleDuties {
		want["role:"+roleID+":assignment"] = len(duties)
	}
	assert.Equal(t, want, withDuties)
}

func TestNamedRoleAssignmentDuties(t *testing.T) {
	cl := testClient(t, estateConfig(t))
	b := newRoleBuilder(cl)

	for roleID, want := range expectedNamedRoleDuties {
		t.Run(roleID, func(t *testing.T) {
			ents := entitlementsAll(t, b, roleResourceByID(t, cl, roleID))
			require.Len(t, ents, 1)
			ent := ents[0]
			assert.Equal(t, roleAssignmentEntitlement, ent.GetSlug())

			got := []expectedDuty{}
			for _, m := range dutyMappings(t, ent) {
				got = append(got, expectedDuty{m.GetDutyId(), m.GetPrecision()})
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestGeneratedResourcesHaveNoDuties(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	for _, b := range []connectorbuilder.ResourceSyncerV2{
		newRoleBuilder(cl),
		newScopedRoleBuilder(cl),
		newGroupBuilder(cl),
		newProjectBuilder(cl),
	} {
		for _, r := range listAll(t, b, 100) {
			if _, named := expectedNamedRoleDuties[r.GetId().GetResource()]; named {
				continue
			}
			for _, ent := range entitlementsAll(t, b, r) {
				assert.Empty(t, dutyMappings(t, ent), "entitlement %s", ent.GetId())
			}
		}
	}
}

func TestNamedRolesDoNotChangeGeneratedRoles(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	ids := map[string]bool{}
	for _, r := range listAll(t, newRoleBuilder(cl), 100) {
		ids[r.GetId().GetResource()] = true
	}
	assert.Len(t, ids, dc.Roles+len(expectedNamedRoleDuties))
	assert.True(t, ids["role-0000000"])
	assert.True(t, ids["role-0000001"])

	// Scoped roles are only built from generated roles.
	for _, r := range listAll(t, newScopedRoleBuilder(cl), 100) {
		roleID, _, err := parseScopedRoleResourceID(r.GetId().GetResource())
		require.NoError(t, err)
		_, named := expectedNamedRoleDuties[roleID]
		assert.False(t, named, "scoped role %s uses a named role", r.GetId().GetResource())
	}
}

func TestNamedRoleSeededConflictingGrants(t *testing.T) {
	cl := testClient(t, estateConfig(t))
	b := newRoleBuilder(cl)

	holders := map[string][]string{}
	for roleID := range expectedNamedRoleDuties {
		for _, g := range grantsAll(t, b, roleResourceByID(t, cl, roleID)) {
			holders[g.GetPrincipal().GetId().GetResource()] = append(holders[g.GetPrincipal().GetId().GetResource()], roleID)
		}
	}

	assert.ElementsMatch(t, []string{"role-payment-preparer", "role-payment-releaser"}, holders["user-0000000"])
	assert.ElementsMatch(t, []string{"role-developer", "role-release-manager"}, holders["user-0000001"])
	assert.ElementsMatch(t, []string{"role-finance-administrator"}, holders["user-0000002"])
	assert.Len(t, holders, 3)
}

func TestNamedRoleGrantRevoke(t *testing.T) {
	cl := testClient(t, estateConfig(t))
	b := newRoleBuilder(cl)
	ctx := context.Background()

	role := roleResourceByID(t, cl, "role-payment-approver")
	ents := entitlementsAll(t, b, role)
	require.Len(t, ents, 1)
	principal, err := resource.NewResource("User 0000002", userResourceType, "user-0000002")
	require.NoError(t, err)

	hasGrant := func() bool {
		for _, g := range grantsAll(t, b, role) {
			if g.GetPrincipal().GetId().GetResource() == "user-0000002" {
				return true
			}
		}
		return false
	}
	require.False(t, hasGrant())

	_, _, err = b.Grant(ctx, principal, ents[0])
	require.NoError(t, err)
	require.True(t, hasGrant())

	_, err = b.Revoke(ctx, &v2.Grant{Entitlement: ents[0], Principal: principal})
	require.NoError(t, err)
	require.False(t, hasGrant())
}
