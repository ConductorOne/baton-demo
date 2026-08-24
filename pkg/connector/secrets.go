package connector

import (
	"context"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type secretBuilder struct {
	client *client.Client
}

var _ connectorbuilder.ResourceSyncerV2 = (*secretBuilder)(nil)

func (o *secretBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return secretResourceType
}

// credentialType maps the client credential-type string to the proto enum.
func credentialType(s string) v2.SecretTrait_CredentialType {
	switch s {
	case client.CredentialTypeStaticSecret:
		return v2.SecretTrait_CREDENTIAL_TYPE_STATIC_SECRET
	case client.CredentialTypeAsymmetricKey:
		return v2.SecretTrait_CREDENTIAL_TYPE_ASYMMETRIC_KEY
	case client.CredentialTypeCertificate:
		return v2.SecretTrait_CREDENTIAL_TYPE_CERTIFICATE
	default:
		return v2.SecretTrait_CREDENTIAL_TYPE_UNSPECIFIED
	}
}

func secretResource(s *client.Secret, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	traitOpts := []resource.SecretTraitOption{
		resource.WithSecretType(credentialType(s.CredentialType)),
	}
	if s.CredentialDetail != "" {
		traitOpts = append(traitOpts, resource.WithSecretDetail(s.CredentialDetail))
	}
	if s.ExpiresAt != nil {
		traitOpts = append(traitOpts, resource.WithSecretExpiresAt(*s.ExpiresAt))
	}
	if s.LastUsedAt != nil {
		traitOpts = append(traitOpts, resource.WithSecretLastUsedAt(*s.LastUsedAt))
	}
	// Back-reference the owning identity (a service-account user) when present.
	if s.IdentityID != "" {
		identityID, err := resource.NewResourceID(userResourceType, s.IdentityID)
		if err != nil {
			return nil, err
		}
		traitOpts = append(traitOpts, resource.WithSecretIdentityID(identityID))
	}

	return resource.NewResource(
		s.Name,
		secretResourceType,
		s.Id,
		resource.WithSecretTrait(traitOpts...),
		resource.WithParentResourceID(parentResourceID),
		resource.WithResourceCreatedAt(s.CreatedAt),
	)
}

func (o *secretBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	secretList, nextPageToken, err := o.client.ListSecrets(ctx, &ops.PageToken)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Resource
	for _, s := range secretList {
		r, err := secretResource(s, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		ret = append(ret, r)
	}

	return ret, &resource.SyncOpResults{NextPageToken: nextPageToken}, nil
}

func (o *secretBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	s, err := o.client.GetSecret(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	r, err := secretResource(s, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return r, nil, nil
}

func (o *secretBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *secretBuilder) Grants(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func newSecretBuilder(client *client.Client) *secretBuilder {
	return &secretBuilder{client: client}
}
