package orchestrator

import (
	"context"
	"testing"
)

// Остановленный пользователем туннель не поднимается сам на WAN-up: кэш
// Enabled зеркалит запись после PersistStopped (F668). Раньше кэш держал
// Enabled=true с загрузки, и decideWANUp делал ColdStart + PersistRunning.
func TestEnabledMirror_StoppedTunnelStaysDownOnWANUp(t *testing.T) {
	o, op, _, store := newRetryFixture(t)
	op.coldStartErr = nil
	_ = o.executeGroup(context.Background(), bootGroup, "boot")
	_ = o.executeGroup(context.Background(), []Action{
		{Type: ActionStopKernel, Tunnel: "awg15"},
		{Type: ActionPersistStopped, Tunnel: "awg15"},
	}, "stop")

	if st, _ := store.Get("awg15"); st.Enabled {
		t.Fatal("запись осталась Enabled после Stop")
	}
	if acts := decideWANUp(Event{Type: EventWANUp, WANIface: "eth3"}, &o.state); len(acts) != 0 {
		t.Fatalf("WAN-up поднимает остановленный туннель: %+v", acts)
	}
}

// Обратное: туннель, выключенный на загрузке и запущенный потом, на
// WAN-down подвешивается, а на WAN-up возвращается (F668).
func TestEnabledMirror_StartedAfterLoadIsSuspendedOnWANDown(t *testing.T) {
	o, op, _, _ := newRetryFixture(t)
	op.coldStartErr = nil
	o.state.tunnels["awg15"].Enabled = false // загрузка с выключенным туннелем
	_ = o.executeGroup(context.Background(), []Action{
		{Type: ActionColdStartKernel, Tunnel: "awg15"},
		{Type: ActionPersistRunning, Tunnel: "awg15"},
	}, "start")

	if !o.state.tunnels["awg15"].Enabled {
		t.Fatal("кэш Enabled не обновлён после старта")
	}
	if acts := decideWANDown(Event{Type: EventWANDown, WANIface: o.state.tunnels["awg15"].ActiveWAN}, &o.state); len(acts) == 0 {
		t.Fatal("WAN-down не подвешивает запущенный туннель")
	}
}

// Каждое действие, пишущее Enabled в запись, отражается в кэше по отдельности.
func TestEnabledMirror_EachWriterUpdatesCache(t *testing.T) {
	for _, tc := range []struct {
		typ  ActionType
		from bool
		want bool
	}{
		{ActionColdStartKernel, false, true},
		{ActionStartNativeWG, false, true},
		{ActionPersistRunning, false, true},
		{ActionPersistStopped, true, false},
	} {
		o, _, _, _ := newRetryFixture(t)
		o.state.tunnels["awg15"].Enabled = tc.from
		o.updateState(Action{Type: tc.typ, Tunnel: "awg15"})
		if got := o.state.tunnels["awg15"].Enabled; got != tc.want {
			t.Errorf("действие %d: Enabled=%v, want %v", tc.typ, got, tc.want)
		}
	}
}
