package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	configschema "github.com/conductorone/baton-sdk/pkg/config"

	"github.com/conductorone/baton-demo/pkg/connector"
)

var version = "dev"

// This is the primary entrypoint for the connector. It uses the SDK to standardize command line flags.
// By using `cli.NewCmd()` from the SDK allows the SDK to be in charge of the lifecycle of your connector logic.
// You are able to add additional flags and update the configuration in case your connector needs more input from the user
// in order to run.
func main() {
	ctx := context.Background()
	_, cmd, err := configschema.DefineConfiguration(ctx, "baton-demo", getConnector, configuration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version
	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, v *viper.Viper) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)
	cb, err := connector.New(ctx)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	c, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
