package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	config "github.com/conductorone/baton-sdk/pb/c1/config/v1"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/actions"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"google.golang.org/protobuf/types/known/structpb"
)

var addNumbers = &v2.BatonActionSchema{
	Name: "addNumbers",
	Arguments: []*config.Field{
		{
			Name:        "number1",
			DisplayName: "Number 1",
			Field:       &config.Field_IntField{},
			IsRequired:  true,
		},
		{
			Name:        "number2",
			DisplayName: "Number 2",
			Field:       &config.Field_IntField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "sum",
			DisplayName: "Sum",
			Field:       &config.Field_IntField{},
		},
	},
}

var helloIn1Minute = &v2.BatonActionSchema{
	Name: "helloIn1Minute",
	Arguments: []*config.Field{
		{
			Name:        "name",
			DisplayName: "Name",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "hello",
			DisplayName: "Hello",
			Field:       &config.Field_StringField{},
		},
	},
}

var disableAccount = &v2.BatonActionSchema{
	Name: "disableAccount",
	Arguments: []*config.Field{
		{
			Name:        "accountId",
			DisplayName: "Account ID",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "success",
			DisplayName: "Success",
			Field:       &config.Field_BoolField{},
		},
	},
	ActionType: []v2.ActionType{
		v2.ActionType_ACTION_TYPE_ACCOUNT,
		v2.ActionType_ACTION_TYPE_ACCOUNT_DISABLE,
	},
}

var enableAccount = &v2.BatonActionSchema{
	Name: "enableAccount",
	Arguments: []*config.Field{
		{
			Name:        "accountId",
			DisplayName: "Account ID",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "success",
			DisplayName: "Success",
			Field:       &config.Field_BoolField{},
		},
	},
	ActionType: []v2.ActionType{
		v2.ActionType_ACTION_TYPE_ACCOUNT,
		v2.ActionType_ACTION_TYPE_ACCOUNT_ENABLE,
	},
}

var updateUserAttrsActionSchema = &v2.BatonActionSchema{
	Name: "update_user_attrs",
	Arguments: []*config.Field{
		{
			Name:        "resource_type",
			DisplayName: "Resource Type",
			Description: "The type of the resource to update.",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
		{
			Name:        "resource_id",
			DisplayName: "Resource ID",
			Description: "The ID of the user to update.",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
		{
			Name:        "attrs",
			DisplayName: "Attributes",
			Description: "The updated attribute data.",
			Field:       &config.Field_StringMapField{},
			IsRequired:  true,
		},
		{
			Name:        "attrs_update_mask",
			DisplayName: "Attributes Update Mask",
			Description: "The attributes to update.",
			Field:       &config.Field_StringSliceField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "success",
			DisplayName: "Success",
			Description: "Whether the account was updated successfully",
			Field:       &config.Field_BoolField{},
		},
	},
	ActionType: []v2.ActionType{
		v2.ActionType_ACTION_TYPE_ACCOUNT,
		v2.ActionType_ACTION_TYPE_ACCOUNT_UPDATE_PROFILE,
	},
}

func (d *Demo) RegisterActionManager(ctx context.Context) (connectorbuilder.CustomActionManager, error) {
	actionManager := actions.NewActionManager(ctx)

	// addNumbers action
	err := actionManager.RegisterAction(ctx, addNumbers.Name, addNumbers, d.addNumbers)
	if err != nil {
		return nil, err
	}

	// helloIn1Minute action
	err = actionManager.RegisterAction(ctx, helloIn1Minute.Name, helloIn1Minute, d.helloIn1Minute)
	if err != nil {
		return nil, err
	}

	err = actionManager.RegisterAction(ctx, enableAccount.Name, enableAccount, d.enableAccount)
	if err != nil {
		return nil, err
	}

	err = actionManager.RegisterAction(ctx, disableAccount.Name, disableAccount, d.disableAccount)
	if err != nil {
		return nil, err
	}

	err = actionManager.RegisterAction(ctx, updateUserAttrsActionSchema.Name, updateUserAttrsActionSchema, d.updateUserAttributes)
	if err != nil {
		return nil, err
	}

	return actionManager, nil
}

func (d *Demo) addNumbers(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	number1, ok := args.Fields["number1"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument number1")
	}
	number2, ok := args.Fields["number2"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument number2")
	}

	if number1.GetNumberValue() > 100 || number2.GetNumberValue() > 100 {
		return nil, nil, fmt.Errorf("numbers must be less than 100")
	}

	sum := number1.GetNumberValue() + number2.GetNumberValue()
	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"sum": structpb.NewNumberValue(sum),
		},
	}
	return response, nil, nil
}

func (d *Demo) helloIn1Minute(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	name, ok := args.Fields["name"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument name")
	}

	// Sleep for 1 minute
	time.Sleep(1 * time.Minute)

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"hello": structpb.NewStringValue("Hello, " + name.GetStringValue() + "!"),
		},
	}
	return response, nil, nil
}

func (d *Demo) enableAccount(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	accountId, ok := args.Fields["accountId"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument accountId")
	}

	l.Info("enabling account", zap.String("accountId", accountId.GetStringValue()))
	user, err := d.client.GetUser(ctx, accountId.GetStringValue())
	if err != nil {
		l.Error("error getting user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update the user's enabled status to true
	user.Enabled = true
	err = d.client.UpdateUser(ctx, user)
	if err != nil {
		l.Error("error updating user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to enable account: %w", err)
	}

	l.Info("enabled account", zap.String("accountId", accountId.GetStringValue()))

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(true),
		},
	}
	return response, nil, nil
}

func (d *Demo) updateUserAttributes(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	resourceType, ok := args.Fields["resource_type"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument resource_type")
	}

	if resourceType.GetStringValue() != "user" {
		return nil, nil, fmt.Errorf("resource type must be user")
	}

	resourceID, ok := args.Fields["resource_id"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument resource_id")
	}
	userID := resourceID.GetStringValue()

	attrsField, ok := args.Fields["attrs"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument attrs")
	}
	attrs := make(map[string]string)
	for k, v := range attrsField.GetStructValue().Fields {
		attrs[k] = v.GetStringValue()
	}

	attrsUpdateMaskList, ok := args.Fields["attrs_update_mask"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument attrs_update_mask")
	}

	var attrsUpdateMask []string
	if attrsUpdateMaskList != nil {
		for _, v := range attrsUpdateMaskList.GetListValue().Values {
			attrsUpdateMask = append(attrsUpdateMask, v.GetStringValue())
		}
	}

	user, err := d.client.GetUser(ctx, userID)
	if err != nil {
		l.Error("error getting user", zap.Error(err))
		return nil, nil, err
	}

	l.Info("updating user attributes", zap.String("user_id", userID))

	for _, attrName := range attrsUpdateMask {
		l.Info("updating user attribute",
			zap.String("user_id", userID),
			zap.String("attr_name", attrName),
			zap.String("attr_value", attrs[attrName]),
			zap.String("old_value", user.Attrs[attrName]),
		)
		if v, ok := attrs[attrName]; ok {
			user.Attrs[attrName] = v
		}
	}

	err = d.client.UpdateUser(ctx, user)
	if err != nil {
		l.Error("error updating user", zap.Error(err))
		return nil, nil, err
	}

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(true),
		},
	}

	return response, nil, nil
}

func (d *Demo) disableAccount(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	accountId, ok := args.Fields["accountId"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument accountId")
	}

	l.Info("disabling account", zap.String("accountId", accountId.GetStringValue()))
	user, err := d.client.GetUser(ctx, accountId.GetStringValue())
	if err != nil {
		l.Error("error getting user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update the user's enabled status to false
	user.Enabled = false
	err = d.client.UpdateUser(ctx, user)
	if err != nil {
		l.Error("error updating user", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to disable account: %w", err)
	}

	l.Info("disabled account", zap.String("accountId", accountId.GetStringValue()))

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(true),
		},
	}
	return response, nil, nil
}
