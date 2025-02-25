package connector

import (
	"context"
	"fmt"
	config "github.com/conductorone/baton-sdk/pb/c1/config/v1"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"google.golang.org/protobuf/types/known/structpb"
)

type ActionManager interface {
	ListActionSchemas(ctx context.Context) ([]*v2.BatonActionSchema, annotations.Annotations, error)
	GetActionSchema(ctx context.Context, name string) (*v2.BatonActionSchema, annotations.Annotations, error)
	InvokeAction(ctx context.Context, name string, args *structpb.Struct) (string, v2.BatonActionStatus, *structpb.Struct, annotations.Annotations, error)
	GetActionStatus(ctx context.Context, id string) (v2.BatonActionStatus, annotations.Annotations, error)
}

var ActionSchemas = map[string]*v2.BatonActionSchema{
	"barnacleFn": {
		Name: "barnacleFn",
		Arguments: []*config.Field{
			{
				Name:        "barnacleMe",
				DisplayName: "Barnacle Me",
				Field:       &config.Field_StringField{},
				IsRequired:  true,
			},
		},
		ReturnTypes: []*config.Field{
			{
				Name:        "barnacledResponse",
				DisplayName: "Barnacled response",
				Field:       &config.Field_StringField{},
			},
		},
	},
}

func (d *Demo) ListActionSchemas(ctx context.Context) ([]*v2.BatonActionSchema, annotations.Annotations, error) {
	var rv []*v2.BatonActionSchema
	for _, v := range ActionSchemas {
		rv = append(rv, v)
	}
	return rv, nil, nil
}

func (d *Demo) GetActionSchema(ctx context.Context, name string) (*v2.BatonActionSchema, annotations.Annotations, error) {
	rv, ok := ActionSchemas[name]
	if !ok {
		return nil, nil, fmt.Errorf("action schema %s not found", name)
	}

	return rv, nil, nil
}

func (d *Demo) InvokeAction(ctx context.Context, name string, args *structpb.Struct) (string, v2.BatonActionStatus, *structpb.Struct, annotations.Annotations, error) {
	switch name {
	case "barnacleFn":
		barnacleMe, ok := args.Fields["barnacleMe"]
		response := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"barnacledResponse": structpb.NewStringValue("Blistering barnacles! " + barnacleMe.GetStringValue()),
			},
		}

		if !ok {
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("missing required argument barnacleMe")
		}
		return name, v2.BatonActionStatus_BATON_ACTION_STATUS_COMPLETE, response, nil, nil
	default:
		return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("action %s not found", name)
	}
}

func (d *Demo) GetActionStatus(ctx context.Context, id string) (v2.BatonActionStatus, annotations.Annotations, error) {
	// TODO: Implement this
	return v2.BatonActionStatus_BATON_ACTION_STATUS_COMPLETE, nil, nil
}
