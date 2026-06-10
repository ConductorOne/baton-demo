package connector

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

func apiKeyOptions() *v2.LocalCredentialOptions {
	o := &v2.LocalCredentialOptions{}
	o.SetApiKey(&v2.LocalCredentialOptions_ApiKey{})
	return o
}

func keypairOptions() *v2.LocalCredentialOptions {
	o := &v2.LocalCredentialOptions{}
	o.SetKeypair(&v2.LocalCredentialOptions_Keypair{})
	return o
}

func tokenOptions() *v2.LocalCredentialOptions {
	o := &v2.LocalCredentialOptions{}
	o.SetToken(&v2.LocalCredentialOptions_Token{})
	return o
}

// assertDemoMaterial checks that vended material is obviously fake and never
// looks like a live credential.
func assertDemoMaterial(t *testing.T, ptds []*v2.PlaintextData) {
	t.Helper()
	require.NotEmpty(t, ptds)
	for _, ptd := range ptds {
		assert.True(t, bytes.HasPrefix(ptd.GetBytes(), []byte("demo-")),
			"material %q is not demo- prefixed", string(ptd.GetBytes()))
	}
}

func TestIssueCredentialMintsSecret(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)
	ctx := context.Background()

	identity := &v2.ResourceId{ResourceType: userResourceType.Id, Resource: "service-account-0000000"}

	secretRes, ptds, _, err := b.Issue(ctx, identity, apiKeyOptions())
	require.NoError(t, err)

	// Fake material returned for the encryption fan-out.
	assertDemoMaterial(t, ptds)

	// The minted secret is a first-class secret resource bound to the identity.
	require.NotNil(t, secretRes)
	assert.Equal(t, secretResourceType.Id, secretRes.GetId().GetResourceType())
	st := pickSecretTrait(t, secretRes)
	require.NotNil(t, st.GetIdentityId())
	assert.Equal(t, identity.GetResource(), st.GetIdentityId().GetResource())
	assert.Equal(t, userResourceType.Id, st.GetIdentityId().GetResourceType())

	// The new secret persists and shows up on a subsequent sync.
	got, err := cl.GetSecret(ctx, secretRes.GetId().GetResource())
	require.NoError(t, err)
	assert.Equal(t, "service-account-0000000", got.IdentityID)
}

func TestIssueRejectsNonMachineOption(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	identity := &v2.ResourceId{ResourceType: userResourceType.Id, Resource: "service-account-0000000"}
	_, _, _, err := b.Issue(context.Background(), identity, &v2.LocalCredentialOptions{})
	require.Error(t, err)
}

func TestIssueRejectsNonUserIdentity(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	identity := &v2.ResourceId{ResourceType: secretResourceType.Id, Resource: "secret-0000000"}
	_, _, _, err := b.Issue(context.Background(), identity, apiKeyOptions())
	require.Error(t, err)
}

func TestIssueCapabilityDetailsAdvertisesMachineOptions(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	details, _, err := b.IssueCapabilityDetails(context.Background())
	require.NoError(t, err)
	assert.Contains(t, details.GetSupportedCredentialOptions(), v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_API_KEY)
	assert.Contains(t, details.GetSupportedCredentialOptions(), v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_KEYPAIR)
	assert.Contains(t, details.GetSupportedCredentialOptions(), v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_TOKEN)
}

func TestCreateAccountMachineCredential(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	accountInfo := &v2.AccountInfo{
		Login:  "svc-bot",
		Emails: []*v2.AccountInfo_Email{{Address: "svc-bot@svc.example.com", IsPrimary: true}},
	}

	resp, ptds, _, err := b.CreateAccount(context.Background(), accountInfo, tokenOptions())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assertDemoMaterial(t, ptds)
	// Token material is named "token".
	assert.Equal(t, "token", ptds[0].GetName())
}

func TestRotateMachineCredential(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	resourceID := &v2.ResourceId{ResourceType: userResourceType.Id, Resource: "service-account-0000000"}
	ptds, _, err := b.Rotate(context.Background(), resourceID, keypairOptions())
	require.NoError(t, err)
	assertDemoMaterial(t, ptds)
	assert.Equal(t, "private_key", ptds[0].GetName())
}

func TestCreateAccountCapabilityAdvertisesMachineOptions(t *testing.T) {
	dc := estateConfig(t)
	cl := testClient(t, dc)
	b := newUserBuilder(cl)

	provDetails, _, err := b.CreateAccountCapabilityDetails(context.Background())
	require.NoError(t, err)
	rotDetails, _, err := b.RotateCapabilityDetails(context.Background())
	require.NoError(t, err)

	for _, opt := range machineCredentialOptions {
		assert.Contains(t, provDetails.GetSupportedCredentialOptions(), opt)
		assert.Contains(t, rotDetails.GetSupportedCredentialOptions(), opt)
	}
}

// TestVendedMaterialNotInError is a guard that the demo prefix marks material as
// non-sensitive demo data (used by reviewers grepping for live-looking tokens).
func TestVendedMaterialUsesDemoPrefix(t *testing.T) {
	material, err := demoToken("apikey")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(material, "demo-apikey-"))
}
