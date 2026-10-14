package diagnostics

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/tunnel"
)

// cycleTunnels — TunnelServiceForDiag, записывающий Restart/SetEnabled.
type cycleTunnels struct {
	diagTunnels
	stopErr, restartErr error
	restarts            int
	enabled             []bool
}

func (c *cycleTunnels) Stop(context.Context, string) error { return c.stopErr }
func (c *cycleTunnels) Restart(context.Context, string) error {
	c.restarts++
	return c.restartErr
}
func (c *cycleTunnels) SetEnabled(_ context.Context, _ string, on bool) error {
	c.enabled = append(c.enabled, on)
	return nil
}

// Упавшая остановка уже записала Enabled=false и сняла маршруты: тест
// обязан вернуть туннель, а не оставить его выключенным (F657).
func TestRestartCycle_StopFailureRestoresTunnel(t *testing.T) {
	svc := &cycleTunnels{stopErr: errors.New("rci")}
	r := &Runner{deps: Deps{TunnelService: svc}}
	res := r.testRestartCycle(context.Background(), TunnelInfo{ID: "awg10", Backend: "kernel"})
	if res.Status != StatusError || svc.restarts != 1 || !strings.Contains(res.Detail, "перезапущен") {
		t.Fatalf("status=%s restarts=%d detail=%q", res.Status, svc.restarts, res.Detail)
	}
	if len(svc.enabled) != 0 {
		t.Fatalf("автозапуск тронут при удачном перезапуске: %v", svc.enabled)
	}
}

// Занятый замок: Stop ничего не сделал — восстанавливать нечего, а Restart
// перебил бы чужую операцию.
func TestRestartCycle_BusyLockDoesNotRestore(t *testing.T) {
	svc := &cycleTunnels{stopErr: tunnel.ErrOperationInProgress}
	r := &Runner{deps: Deps{TunnelService: svc}}
	r.testRestartCycle(context.Background(), TunnelInfo{ID: "awg10", Backend: "kernel"})
	if svc.restarts != 0 || len(svc.enabled) != 0 {
		t.Fatalf("restarts=%d enabled=%v", svc.restarts, svc.enabled)
	}
}

// Не поднялся и перезапуск — автозапуск остаётся включённым (F657).
func TestRestartCycle_RestoreFallsBackToAutostart(t *testing.T) {
	svc := &cycleTunnels{restartErr: errors.New("no wan")}
	r := &Runner{deps: Deps{TunnelService: svc}}
	detail := r.restoreAfterCycle(context.Background(), "awg10")
	if svc.restarts != 1 || len(svc.enabled) != 1 || !svc.enabled[0] || !strings.Contains(detail, "автозапуск") {
		t.Fatalf("restarts=%d enabled=%v detail=%q", svc.restarts, svc.enabled, detail)
	}
}
