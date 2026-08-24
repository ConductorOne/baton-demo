package connector

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conductorone/baton-demo/pkg/client"
	"github.com/conductorone/baton-demo/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

// estateConfig returns a small but complete NHI demo estate config backed by a
// fresh temp database.
func estateConfig(t *testing.T) *config.Demo {
	t.Helper()
	return &config.Demo{
		Users:           3,
		Groups:          2,
		Roles:           2,
		ScopedRoles:     1,
		Projects:        1,
		ServiceAccounts: 3,
		SystemAccounts:  1,
		Secrets:         5,
		UnownedSecrets:  2,
		NhiApps:         3,
		AssumableRoles:  2,
		Agents:          2,
		InitDb:          true,
		DbFileName:      filepath.Join(t.TempDir(), "estate.db"),
	}
}

func testClient(t *testing.T, dc *config.Demo) *client.Client {
	t.Helper()
	cl, err := client.NewClient(context.Background(), dc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cl.Close() })
	return cl
}

// listAll drains every page of a builder's List, exercising the SDK page-token
// pagination path with the given page size.
func listAll(t *testing.T, b connectorbuilder.ResourceSyncerV2, pageSize int) []*v2.Resource {
	t.Helper()
	ctx := context.Background()
	var out []*v2.Resource
	token := ""
	for {
		res, results, err := b.List(ctx, nil, resource.SyncOpAttrs{
			PageToken: pagination.Token{Size: pageSize, Token: token},
		})
		require.NoError(t, err)
		out = append(out, res...)
		if results == nil || results.NextPageToken == "" {
			break
		}
		token = results.NextPageToken
	}
	return out
}

func pickSecretTrait(t *testing.T, r *v2.Resource) *v2.SecretTrait {
	t.Helper()
	st := &v2.SecretTrait{}
	annos := annotations.Annotations(r.GetAnnotations())
	ok, err := annos.Pick(st)
	require.NoError(t, err)
	require.True(t, ok, "resource %s missing SecretTrait", r.GetId().GetResource())
	return st
}

func TestUsersEmitAccountTypes(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	users := listAll(t, newUserBuilder(cl), 100)

	counts := map[v2.UserTrait_AccountType]int{}
	for _, r := range users {
		ut, err := resource.GetUserTrait(r)
		require.NoError(t, err)
		counts[ut.GetAccountType()]++
	}

	assert.Equal(t, dc.Users, counts[v2.UserTrait_ACCOUNT_TYPE_HUMAN])
	assert.Equal(t, dc.ServiceAccounts, counts[v2.UserTrait_ACCOUNT_TYPE_SERVICE])
	assert.Equal(t, dc.SystemAccounts, counts[v2.UserTrait_ACCOUNT_TYPE_SYSTEM])
}

func TestSecretsEmitSecretTrait(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	secrets := listAll(t, newSecretBuilder(cl), 100)
	require.Len(t, secrets, dc.Secrets+dc.UnownedSecrets)

	owned, unowned := 0, 0
	credTypes := map[v2.SecretTrait_CredentialType]int{}
	for _, r := range secrets {
		assert.Equal(t, secretResourceType.Id, r.GetId().GetResourceType())
		st := pickSecretTrait(t, r)
		assert.NotEqual(t, v2.SecretTrait_CREDENTIAL_TYPE_UNSPECIFIED, st.GetCredentialType())
		credTypes[st.GetCredentialType()]++
		require.NotNil(t, r.GetCreatedAt())
		if st.GetIdentityId() != nil {
			owned++
			// Owned secrets back-reference a user (service account) resource.
			assert.Equal(t, userResourceType.Id, st.GetIdentityId().GetResourceType())
		} else {
			unowned++
		}
	}
	assert.Equal(t, dc.Secrets, owned)
	assert.Equal(t, dc.UnownedSecrets, unowned)
	// The estate exercises all three credential types.
	assert.Positive(t, credTypes[v2.SecretTrait_CREDENTIAL_TYPE_STATIC_SECRET])
	assert.Positive(t, credTypes[v2.SecretTrait_CREDENTIAL_TYPE_ASYMMETRIC_KEY])
	assert.Positive(t, credTypes[v2.SecretTrait_CREDENTIAL_TYPE_CERTIFICATE])
}

func TestSecretsPagination(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	// A small page size forces multiple pages via the SDK page token.
	secrets := listAll(t, newSecretBuilder(cl), 2)
	assert.Len(t, secrets, dc.Secrets+dc.UnownedSecrets)

	// IDs must be unique across pages (no duplicates / no gaps).
	seen := map[string]bool{}
	for _, r := range secrets {
		id := r.GetId().GetResource()
		assert.False(t, seen[id], "duplicate secret %s across pages", id)
		seen[id] = true
	}
}

func TestNHIAppsEmitNonHumanIdentityTrait(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	apps := listAll(t, newNHIAppBuilder(cl), 100)
	require.Len(t, apps, dc.NhiApps)

	nhiTypes := map[v2.NonHumanIdentityTrait_NhiType]int{}
	for _, r := range apps {
		assert.Equal(t, nhiAppResourceType.Id, r.GetId().GetResourceType())
		_, err := resource.GetAppTrait(r)
		require.NoError(t, err)
		nhi, err := resource.GetNonHumanIdentityTrait(r)
		require.NoError(t, err)
		assert.NotEqual(t, v2.NonHumanIdentityTrait_NHI_TYPE_UNSPECIFIED, nhi.GetNhiType())
		assert.NotEmpty(t, nhi.GetNhiDetail())
		nhiTypes[nhi.GetNhiType()]++
	}
	assert.Positive(t, nhiTypes[v2.NonHumanIdentityTrait_NHI_TYPE_APP_REGISTRATION])
	assert.Positive(t, nhiTypes[v2.NonHumanIdentityTrait_NHI_TYPE_MANAGED_IDENTITY])
}

func TestAssumableRolesEmitNonHumanIdentityTrait(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	roles := listAll(t, newAssumableRoleBuilder(cl), 100)
	require.Len(t, roles, dc.AssumableRoles)

	for _, r := range roles {
		assert.Equal(t, assumableRoleResourceType.Id, r.GetId().GetResourceType())
		_, err := resource.GetRoleTrait(r)
		require.NoError(t, err)
		nhi, err := resource.GetNonHumanIdentityTrait(r)
		require.NoError(t, err)
		assert.Equal(t, v2.NonHumanIdentityTrait_NHI_TYPE_ASSUMABLE_ROLE, nhi.GetNhiType())
	}
}

func TestAgentsEmitAgentTrait(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)

	agents := listAll(t, newAgentBuilder(cl), 100)
	require.Len(t, agents, dc.Agents)

	for _, r := range agents {
		assert.Equal(t, agentResourceType.Id, r.GetId().GetResourceType())
		at, err := resource.GetAgentTrait(r)
		require.NoError(t, err)
		require.NotNil(t, r.GetStatus())
		assert.NotEqual(t, v2.Status_RESOURCE_STATUS_UNSPECIFIED, r.GetStatus().GetStatus())
		// Each agent authenticates as a service-account user.
		require.NotNil(t, at.GetIdentityResourceId())
		assert.Equal(t, userResourceType.Id, at.GetIdentityResourceId().GetResourceType())
	}
}
