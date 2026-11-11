package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/health"
)

type fakePinger struct {
	err error
}

func (p fakePinger) Ping(context.Context) error {
	return p.err
}

func TestReadinessReturnsReadyWhenDatabasePingSucceeds(t *testing.T) {
	result := health.NewService(fakePinger{}).Readiness(context.Background())

	if !result.Ready {
		t.Fatal("ready = false, want true")
	}
	if result.Database != "ok" {
		t.Fatalf("database = %q, want ok", result.Database)
	}
}

func TestReadinessReturnsUnavailableWithoutLeakingDriverError(t *testing.T) {
	secretError := errors.New("dial postgres://user:super-secret@db/private failed")
	result := health.NewService(fakePinger{err: secretError}).Readiness(context.Background())

	if result.Ready {
		t.Fatal("ready = true, want false")
	}
	if result.Database != "unavailable" {
		t.Fatalf("database = %q, want unavailable", result.Database)
	}
	if result.Error != "" {
		t.Fatalf("error = %q, want empty safe response", result.Error)
	}
}
