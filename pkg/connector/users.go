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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/conductorone/baton-demo/pkg/client"
	config "github.com/conductorone/baton-sdk/pb/c1/config/v1"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/actions"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/crypto"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

var updateUserProfileSchema = &v2.BatonActionSchema{
	Name:        "update_profile",
	DisplayName: "Update User Profile",
	Description: "Update a user's profile attributes",
	ActionType:  []v2.ActionType{v2.ActionType_ACTION_TYPE_ACCOUNT_UPDATE_PROFILE},

	Arguments: []*config.Field{
		{
			Name:        "user_id",
			DisplayName: "User",
			Description: "The user to update",
			IsRequired:  true,
			Field: &config.Field_ResourceIdField{
				ResourceIdField: &config.ResourceIdField{
					Rules: &config.ResourceIDRules{
						AllowedResourceTypeIds: []string{"user"},
					},
				},
			},
		},
		{
			Name:        "name",
			DisplayName: "Name",
			Field:       &config.Field_StringField{},
		},
		{
			Name:        "email",
			DisplayName: "Email",
			Field:       &config.Field_StringField{},
		},
		{
			Name:        "custom_attributes",
			DisplayName: "Custom Attributes",
			Description: "Additional custom attributes to set on the user",
			Field:       &config.Field_StringMapField{},
		},
	},

	ReturnTypes: []*config.Field{
		{
			Name:        "updated_user",
			DisplayName: "Updated User",
			Field:       &config.Field_ResourceField{},
		},
	},
}

type userBuilder struct {
	client *client.Client
}

var _ connectorbuilder.AccountManagerLimited = &userBuilder{}
var _ connectorbuilder.CredentialManagerLimited = &userBuilder{}
var _ connectorbuilder.ResourceActionProvider = &userBuilder{}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func (o *userBuilder) ResourceActions(ctx context.Context, registry actions.ActionRegistry) error {
	return registry.Register(ctx, updateUserProfileSchema, o.updateUserProfile)
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

	traits := []resource.UserTraitOption{
		resource.WithEmail(u.Email, true),
		resource.WithUserLogin(u.Id),
		resource.WithDetailedStatus(status, statusMessage),
		resource.WithEmployeeID(u.Id),
		resource.WithAccountType(v2.UserTrait_ACCOUNT_TYPE_HUMAN),
		resource.WithUserProfile(attrs),
	}
	return resource.NewUserResource(
		u.Name,
		userResourceType,
		u.Id,
		traits,
		resource.WithParentResourceID(parentResourceID),
		resource.WithAnnotation(
			&v2.Aliases{
				Id: []*v2.IdAlias{
					{
						Id: client.OldId(u.Id),
					},
				},
			},
		),
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, ops resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	users, nextPageToken, err := o.client.ListUsers(ctx, &ops.PageToken)
	if err != nil {
		return nil, nil, err
	}

	var ret []*v2.Resource
	for _, u := range users {
		userResource, err := userResource(u, parentResourceID)
		if err != nil {
			return nil, nil, err
		}

		ret = append(ret, userResource)
	}

	return ret, &resource.SyncOpResults{NextPageToken: nextPageToken}, nil
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
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, ops resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, ops resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	return nil, nil, nil
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

func (o *userBuilder) updateUserProfile(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	// Use global logger since action handlers receive a fresh context without the logger attached
	l := zap.L()

	// Extract user_id (required)
	userIDField, ok := args.Fields["user_id"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument user_id")
	}
	userID := userIDField.GetStringValue()

	// Fetch the user
	user, err := o.client.GetUser(ctx, userID)
	if err != nil {
		l.Error("error getting user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	l.Info("updateUserProfile: updating user", zap.String("user", user.Name), zap.String("user_id", user.Id), zap.Any("args", args))

	// Update name if provided
	if nameField, ok := args.Fields["name"]; ok && nameField.GetStringValue() != "" {
		user.Name = nameField.GetStringValue()
	}

	// Update email if provided
	if emailField, ok := args.Fields["email"]; ok && emailField.GetStringValue() != "" {
		user.Email = emailField.GetStringValue()
	}

	// Merge custom_attributes if provided
	if customAttrsField, ok := args.Fields["custom_attributes"]; ok && customAttrsField.GetStructValue() != nil {
		if user.Attrs == nil {
			user.Attrs = make(map[string]string)
		}
		for k, v := range customAttrsField.GetStructValue().Fields {
			user.Attrs[k] = v.GetStringValue()
		}
	}

	// Save the updated user
	err = o.client.UpdateUser(ctx, user)
	if err != nil {
		l.Error("error updating user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Build the updated user resource
	updatedResource, err := userResource(user, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build updated user resource: %w", err)
	}

	// Convert resource to structpb for return
	resourceStruct, err := structpb.NewStruct(map[string]interface{}{
		"updated_user": map[string]interface{}{
			"id":           updatedResource.Id.Resource,
			"display_name": updatedResource.DisplayName,
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create response struct: %w", err)
	}

	return resourceStruct, nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
