package connector

import (
	"context"
	"fmt"
	"time"

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
	accountId, ok := args.Fields["accountId"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument accountId")
	}

	fmt.Println("Enabling account", accountId.GetStringValue())

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(true),
		},
	}
	return response, nil, nil	
}

func (d *Demo) disableAccount(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	accountId, ok := args.Fields["accountId"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required argument accountId")
	}
	
	fmt.Println("Disabling account", accountId.GetStringValue())

	response := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(true),
		},
	}
	return response, nil, nil
}
