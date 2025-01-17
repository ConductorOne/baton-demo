package cli

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	c1_lambda_grpc "github.com/ductone/c1-lambda/pkg/grpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/conductorone/baton-sdk/internal/connector"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/logging"
)

func MakeLambdaServerCommand(
	ctx context.Context,
	name string,
	v *viper.Viper,
	confschema field.Configuration,
	getconnector GetConnectorFunc,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := v.BindPFlags(cmd.Flags())
		if err != nil {
			return err
		}

		runCtx, err := initLogger(
			ctx,
			name,
			logging.WithLogFormat(v.GetString("log-format")),
			logging.WithLogLevel(v.GetString("log-level")),
		)
		if err != nil {
			return err
		}

		// validate required fields and relationship constraints
		// TODO(morgabra/kans): We need to fetch our config before we can instantiate a connector...
		if err := field.Validate(confschema, v); err != nil {
			return err
		}

		c, err := getconnector(runCtx, v)
		if err != nil {
			return err
		}

		opts := &connector.RegisterOps{
			Ratelimiter:         nil,
			ProvisioningEnabled: v.GetBool("provisioning"),
			TicketingEnabled:    v.GetBool("ticketing"),
		}

		s := c1_lambda_grpc.NewServer(nil)
		connector.Register(ctx, s, c, opts)

		lambda.Start(s.Handler)
		return nil
	}
}
