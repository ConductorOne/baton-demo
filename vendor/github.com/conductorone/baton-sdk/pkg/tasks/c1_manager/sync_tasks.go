package c1_manager

import (
	"context"
	"time"

	sdkSync "github.com/conductorone/baton-sdk/pkg/sync"
	"github.com/conductorone/baton-sdk/pkg/tasks"
	"github.com/conductorone/baton-sdk/pkg/types"
)

func (c *c1TaskManager) handleLocalFileSync(ctx context.Context, cc types.ConnectorClient, t *tasks.LocalFileSync) error {
	syncer, err := sdkSync.NewSyncer(ctx, cc, t.DbPath)
	if err != nil {
		return err
	}

	err = syncer.Sync(ctx)
	if err != nil {
		return err
	}

	err = syncer.Close(ctx)
	if err != nil {
		return err
	}

	// FIXME(jirwin): temp delay to test heartbeating and such
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second * 5):
	}

	return nil
}
