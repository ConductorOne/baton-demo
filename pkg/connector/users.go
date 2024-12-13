package connector

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"

	"github.com/conductorone/baton-demo/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/crypto"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	users, err := o.client.ListUsers(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, u := range users {
		userResource, err := sdkResource.NewUserResource(u.Name, userResourceType, u.Id, []sdkResource.UserTraitOption{
			sdkResource.WithEmail(u.Email, true),
		}, sdkResource.WithParentResourceID(parentResourceID))
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, userResource)
	}

	return ret, "", nil, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (r *userBuilder) RotateCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsCredentialRotation, annotations.Annotations, error) {
	return &v2.CredentialDetailsCredentialRotation{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD, v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD},
		PreferredCredentialOption:  v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
	}, nil, nil
}

func (o *userBuilder) Rotate(ctx context.Context, resourceId *v2.ResourceId, credentialOptions *v2.CredentialOptions) ([]*v2.PlaintextData, annotations.Annotations, error) {
	if resourceId.ResourceType != roleResourceType.Id {
		return nil, nil, fmt.Errorf("baton-postgres: non-role/user resource passed to rotate credentials")
	}

	user, err := o.client.GetUser(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}

	var plainTextPassword string
	var ptd *v2.PlaintextData
	if credentialOptions.GetRandomPassword() != nil {
		plainTextPassword, err = crypto.GeneratePassword(credentialOptions)
		if err != nil {
			return nil, nil, err
		}
		ptd = &v2.PlaintextData{
			Name:  "password",
			Bytes: []byte(plainTextPassword),
		}
	}

	err = o.client.ChangePassword(ctx, user.Id, plainTextPassword)
	if err != nil {
		return nil, nil, err
	}

	return []*v2.PlaintextData{ptd}, nil, nil
}

func (o *userBuilder) makeResource(ctx context.Context, user *client.User) (*v2.Resource, error) {
	return sdkResource.NewUserResource(user.Name, userResourceType, user.Id, nil)
}

func (o *userBuilder) CreateAccountCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD, v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD},
		PreferredCredentialOption:  v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
	}, nil, nil
}

func (o *userBuilder) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	credentialOptions *v2.CredentialOptions,
) (connectorbuilder.CreateAccountResponse, []*v2.PlaintextData, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	var plainTextPassword string
	var err error
	var ptd *v2.PlaintextData
	if credentialOptions.GetRandomPassword() != nil {
		l.Info("Generating random password")
		plainTextPassword, err = crypto.GeneratePassword(credentialOptions)
		if err != nil {
			return nil, nil, nil, err
		}
		ptd = &v2.PlaintextData{
			Name:  "password",
			Bytes: []byte(plainTextPassword),
		}
	} else {
		l.Info("No password generated")
	}

	createdUser, err := o.client.CreateUser(ctx, accountInfo.Login, accountInfo.Emails[0].String(), plainTextPassword)
	if err != nil {
		return nil, nil, nil, err
	}

	resource, err := o.makeResource(ctx, createdUser)
	if err != nil {
		return nil, nil, nil, err
	}

	return &v2.CreateAccountResponse_SuccessResult{
		Resource: resource,
	}, []*v2.PlaintextData{ptd}, nil, nil
}

func (o *userBuilder) Create(ctx context.Context, resource *v2.Resource) (*v2.Resource, annotations.Annotations, error) {
	return nil, nil, fmt.Errorf("baton-demo: role creation not supported")
}

func (o *userBuilder) Delete(ctx context.Context, resourceId *v2.ResourceId) (annotations.Annotations, error) {
	if resourceId.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-demo: non-user resource passed to role delete")
	}

	pgRole, err := o.client.GetUser(ctx, resourceId.Resource)
	if err != nil {
		return nil, err
	}

	err = o.client.DeleteUser(ctx, pgRole.Name)
	return nil, err
}

func (o *userBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	return nil, nil, nil
}

func (o *userBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	return nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
