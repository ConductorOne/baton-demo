package connector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

var _ connectorbuilder.AccountManager = &userBuilder{}
var _ connectorbuilder.CredentialManager = &userBuilder{}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func userResource(u *client.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	// Determine the user status based on the enabled field
	var status v2.UserTrait_Status_Status
	var statusMessage string
	if u.Enabled {
		status = v2.UserTrait_Status_STATUS_ENABLED
		statusMessage = "Enabled"
	} else {
		status = v2.UserTrait_Status_STATUS_DISABLED
		statusMessage = "Disabled"
	}

	attrs := make(map[string]any)
	for k, v := range u.Attrs {
		attrs[k] = v
	}
	// Add the enabled status to the profile
	attrs["enabled"] = u.Enabled

	traits := []sdkResource.UserTraitOption{
		sdkResource.WithEmail(u.Email, true),
		sdkResource.WithUserLogin(u.Id),
		sdkResource.WithDetailedStatus(status, statusMessage),
		sdkResource.WithEmployeeID(u.Id),
		sdkResource.WithAccountType(v2.UserTrait_ACCOUNT_TYPE_HUMAN),
		sdkResource.WithUserProfile(attrs),
	}
	return sdkResource.NewUserResource(
		u.Name,
		userResourceType,
		u.Id,
		traits,
		sdkResource.WithParentResourceID(parentResourceID),
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	users, nextPageToken, err := o.client.ListUsers(ctx, pToken)
	if err != nil {
		return nil, "", nil, err
	}

	var ret []*v2.Resource
	for _, u := range users {
		userResource, err := userResource(u, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}

		ret = append(ret, userResource)
	}

	return ret, nextPageToken, nil, nil
}

func (o *userBuilder) Get(ctx context.Context, resourceId *v2.ResourceId, parentResourceId *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	user, err := o.client.GetUser(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}
	resource, err := userResource(user, parentResourceId)
	if err != nil {
		return nil, nil, err
	}
	return resource, nil, nil
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
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_ENCRYPTED_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
	}, nil, nil
}

func (o *userBuilder) Rotate(ctx context.Context, resourceId *v2.ResourceId, credentialOptions *v2.LocalCredentialOptions) ([]*v2.PlaintextData, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if resourceId.ResourceType != userResourceType.Id {
		return nil, nil, status.Error(codes.InvalidArgument, "baton-demo: non-user resource passed to rotate credentials")
	}

	if credentialOptions.GetRandomPassword() == nil && credentialOptions.GetPlaintextPassword() == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "baton-demo: no password provided")
	}

	user, err := o.client.GetUser(ctx, resourceId.Resource)
	if err != nil {
		return nil, nil, err
	}

	plainTextPassword, err := crypto.GeneratePassword(ctx, credentialOptions)
	if err != nil {
		return nil, nil, err
	}
	ptd := &v2.PlaintextData{
		Name:  "password",
		Bytes: []byte(plainTextPassword),
	}

	l.Info("Changing password", zap.String("user_id", user.Id), zap.String("password", plainTextPassword))
	err = o.client.ChangePassword(ctx, user.Id, plainTextPassword)
	if err != nil {
		return nil, nil, err
	}

	return []*v2.PlaintextData{ptd}, nil, nil
}

func (o *userBuilder) CreateAccountCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_ENCRYPTED_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_RANDOM_PASSWORD,
	}, nil, nil
}

func (o *userBuilder) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	credentialOptions *v2.LocalCredentialOptions,
) (connectorbuilder.CreateAccountResponse, []*v2.PlaintextData, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if credentialOptions.GetRandomPassword() == nil && credentialOptions.GetPlaintextPassword() == nil {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-demo: no password provided")
	}

	l.Info("Generating password")
	plainTextPassword, err := crypto.GeneratePassword(ctx, credentialOptions)
	if err != nil {
		return nil, nil, nil, err
	}
	ptd := &v2.PlaintextData{
		Name:  "password",
		Bytes: []byte(plainTextPassword),
	}
	l.Info("Creating user", zap.String("user_id", accountInfo.Login), zap.String("password", plainTextPassword))

	if len(accountInfo.Emails) == 0 {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-demo: no email provided")
	}
	createdUser, err := o.client.CreateUser(ctx, accountInfo.Login, accountInfo.Emails[0].Address, plainTextPassword)
	if err != nil {
		return nil, nil, nil, err
	}

	resource, err := userResource(createdUser, nil)
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

	user, err := o.client.GetUser(ctx, resourceId.Resource)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User already deleted.
			return nil, nil
		}
		return nil, err
	}

	err = o.client.DeleteUser(ctx, user.Id)
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
