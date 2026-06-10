package connector

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

// The demo connector is the reference implementation of the machine-credential
// vending surface (baton-sdk#941): the new ApiKey/Keypair/Token CredentialOptions
// arms, the CredentialIssuer interface, and the CAPABILITY_CREDENTIAL_ISSUE
// advertisement. All vended material is obviously fake (demo- prefixed) and is
// returned as PlaintextData so the builder's encryption fan-out is exercised; it
// is never logged.

// userBuilder is a ResourceSyncerV2 that opts in to credential issuance by
// implementing CredentialIssuerLimited; the builder registers it automatically.
var _ connectorbuilder.CredentialIssuerLimited = (*userBuilder)(nil)

// machineCredentialOptions lists the machine-credential arms the demo vends.
var machineCredentialOptions = []v2.CapabilityDetailCredentialOption{
	v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_API_KEY,
	v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_KEYPAIR,
	v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_TOKEN,
}

// demoToken returns obviously-fake credential material with a demo- prefix.
func demoToken(prefix string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "demo-" + prefix + "-" + hex.EncodeToString(buf), nil
}

// machineCredentialMaterial generates fake material for a machine-credential
// option arm and reports the SecretTrait classification of what was minted. It
// returns (nil, ...) when the options carry no machine-credential arm.
func machineCredentialMaterial(opts *v2.LocalCredentialOptions) ([]*v2.PlaintextData, string, string, error) {
	switch {
	case opts.GetApiKey() != nil:
		material, err := demoToken("apikey")
		if err != nil {
			return nil, "", "", err
		}
		return []*v2.PlaintextData{{Name: "api_key", Bytes: []byte(material)}},
			client.CredentialTypeStaticSecret, "api_key", nil
	case opts.GetKeypair() != nil:
		// Client-side keypair generation: the private key is returned as
		// PlaintextData and the builder encrypts it before it leaves the
		// connector. The demo emits fake material only.
		material, err := demoToken("privatekey")
		if err != nil {
			return nil, "", "", err
		}
		return []*v2.PlaintextData{{Name: "private_key", Bytes: []byte(material)}},
			client.CredentialTypeAsymmetricKey, "keypair", nil
	case opts.GetToken() != nil:
		material, err := demoToken("token")
		if err != nil {
			return nil, "", "", err
		}
		return []*v2.PlaintextData{{Name: "token", Bytes: []byte(material)}},
			client.CredentialTypeStaticSecret, "token", nil
	default:
		return nil, "", "", nil
	}
}

// credentialOptionTTL returns the expiry implied by a machine-credential arm's
// TTL, or nil when none is set.
func credentialOptionTTL(opts *v2.LocalCredentialOptions) *time.Time {
	var d time.Duration
	switch {
	case opts.GetApiKey() != nil:
		d = opts.GetApiKey().GetTtl().AsDuration()
	case opts.GetToken() != nil:
		d = opts.GetToken().GetTtl().AsDuration()
	default:
		return nil
	}
	if d <= 0 {
		return nil
	}
	exp := time.Now().Add(d)
	return &exp
}

// IssueCapabilityDetails advertises the machine-credential options the demo can
// mint via IssueCredential.
func (o *userBuilder) IssueCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsCredentialIssue, annotations.Annotations, error) {
	return &v2.CredentialDetailsCredentialIssue{
		SupportedCredentialOptions: machineCredentialOptions,
		PreferredCredentialOption:  v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_API_KEY,
	}, nil, nil
}

// Issue mints a NEW secret for an existing identity (a service-account user),
// distinct from Rotate's replace-in-place semantics. The new secret is persisted
// (so it shows up on the next sync, carrying a SecretTrait with an identity_id
// back-ref) and the fake material is returned as PlaintextData for the builder
// to encrypt.
func (o *userBuilder) Issue(
	ctx context.Context,
	identityID *v2.ResourceId,
	credentialOptions *v2.LocalCredentialOptions,
) (*v2.Resource, []*v2.PlaintextData, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if identityID.GetResourceType() != userResourceType.Id {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-demo: credentials can only be issued for user identities")
	}

	user, err := o.client.GetUser(ctx, identityID.GetResource())
	if err != nil {
		return nil, nil, nil, err
	}

	plaintexts, credType, detail, err := machineCredentialMaterial(credentialOptions)
	if err != nil {
		return nil, nil, nil, err
	}
	if plaintexts == nil {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-demo: issue requires a machine-credential option (api_key, keypair, or token)")
	}

	now := time.Now()
	secret := &client.Secret{
		Name:             "Issued " + detail + " for " + user.Id,
		CredentialType:   credType,
		CredentialDetail: detail,
		IdentityID:       user.Id,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        credentialOptionTTL(credentialOptions),
	}
	created, err := o.client.CreateSecret(ctx, secret)
	if err != nil {
		return nil, nil, nil, err
	}

	// Do not log vended material — only metadata.
	l.Info("issued credential",
		zap.String("identity_id", user.Id),
		zap.String("secret_id", created.Id),
		zap.String("credential_detail", detail),
	)

	secretRes, err := secretResource(created, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	return secretRes, plaintexts, nil, nil
}
