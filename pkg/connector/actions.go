package connector

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

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
	"addNumbers": {
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
	},
	"openYouTube": {
		Name: "openYouTube",
		Arguments: []*config.Field{
			{
				Name:        "url",
				DisplayName: "YouTube URL",
				Field:       &config.Field_StringField{},
				IsRequired:  false,
			},
		},
		ReturnTypes: []*config.Field{
			{
				Name:        "result",
				DisplayName: "Result",
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
	case "addNumbers":
		number1, ok := args.Fields["number1"]
		if !ok {
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("missing required argument number1")
		}
		number2, ok := args.Fields["number2"]
		if !ok {
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("missing required argument number2")
		}

		if number1.GetNumberValue() > 100 || number2.GetNumberValue() > 100 {
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("numbers must be less than 100")
		}

		sum := number1.GetNumberValue() + number2.GetNumberValue()
		response := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"sum": structpb.NewNumberValue(sum),
			},
		}
		return name, v2.BatonActionStatus_BATON_ACTION_STATUS_COMPLETE, response, nil, nil
	case "openYouTube":
		// Default YouTube URL (Rick Roll)
		youtubeURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

		// Check if a custom URL was provided
		if urlValue, ok := args.Fields["url"]; ok && urlValue.GetStringValue() != "" {
			youtubeURL = urlValue.GetStringValue()
		}

		var cmd *exec.Cmd
		var resultMsg string

		// Check if we're on macOS
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("open", youtubeURL)
			resultMsg = "You've been Rick Rolled!"
		} else {
			// For other platforms, we could add support later
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("opening browser is only supported on macOS")
		}

		err := cmd.Start()
		if err != nil {
			return "", v2.BatonActionStatus_BATON_ACTION_STATUS_FAILED, nil, nil, fmt.Errorf("failed to open browser: %v", err)
		}

		response := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"result": structpb.NewStringValue(resultMsg),
			},
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
