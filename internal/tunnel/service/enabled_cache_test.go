package service

import (
	"context"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/orchestrator"
	"github.com/hoaxisr/awg-manager/internal/storage"
)

// Выключенный автостарт доходит до кэша оркестратора: WAN-up не поднимает
// туннель (F668). Оркестратор без операторов — ColdStart на nil-операторе
// упал бы паникой, так что «действий нет» и есть проверка.
func TestSetEnabled_RefreshesOrchestratorCache(t *testing.T) {
	s, _ := serviceWithStore(t)
	if err := s.store.Create(&storage.AWGTunnel{ID: "awg10", Name: "t", Backend: "kernel", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	orch := orchestrator.New(s.store, nil, nil, nil, nil, nil)
	s.SetOrchestrator(orch)
	orch.RefreshTunnelState("awg10") // кэш с загрузки: Enabled=true

	if err := s.SetEnabled(context.Background(), "awg10", false); err != nil {
		t.Fatal(err)
	}
	if err := orch.HandleEvent(context.Background(), orchestrator.Event{Type: orchestrator.EventWANUp, WANIface: "eth3"}); err != nil {
		t.Fatalf("WAN-up: %v", err)
	}
}
