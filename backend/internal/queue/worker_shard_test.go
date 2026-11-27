package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func TestShardHaltMiddlewareBlocksTasksWhenHalted(t *testing.T) {
	called := false
	handler := shardHaltMiddleware(asynq.HandlerFunc(func(_ context.Context, _ *asynq.Task) error {
		called = true
		return nil
	}))

	util.ApplyRemoteShardState(true, 0x7F)
	defer util.ApplyRemoteShardState(false, 0x00)

	err := handler.ProcessTask(context.Background(), asynq.NewTask(TypeWhatsAppEvent, []byte(`{}`)))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("expected SkipRetry when halted, got %v", err)
	}
	if called {
		t.Fatal("inner handler must not execute while instance is halted")
	}

	util.ApplyRemoteShardState(false, 0x00)
	if err := handler.ProcessTask(context.Background(), asynq.NewTask(TypeWhatsAppEvent, []byte(`{}`))); err != nil {
		t.Fatalf("expected task to run after unhalt, got %v", err)
	}
	if !called {
		t.Fatal("expected inner handler to execute after unhalt")
	}
}
