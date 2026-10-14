package dnsroute

import (
	"sync"

	"github.com/hoaxisr/awg-manager/internal/events"
)

// Logger is the minimal logger interface used by FailoverManager.
type Logger interface {
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// AffectedList describes a DNS route list affected by a tunnel state change.
type AffectedList struct {
	ListID     string
	ListName   string
	FromTunnel string // tunnel interface name — currently active for the list
	ToTunnel   string // tunnel interface name that will become active after failover
}

// AffectedListsLookup returns the DNS route lists that reference a given tunnelID.
// For "switched" action, FromTunnel is the failed tunnel and ToTunnel is the next active.
// For "restored" action, the order is reversed.
type AffectedListsLookup func(tunnelID string, action string) []AffectedList

// FailoverManager tracks which tunnels have failed and need DNS route failover.
// In-memory only — state resets on restart (reconcile rebuilds from primary targets).
type FailoverManager struct {
	mu          sync.RWMutex
	failedSet   map[string]struct{}
	reconcileFn func() error // returns error so we can rollback / retry
	// dirty is a global "best-effort retry" flag, not a per-tunnel retry queue.
	// Set when any reconcile fails; cleared on the next successful reconcile.
	// While dirty, the next event force-retries even on duplicate (alreadyFailed)
	// tunnels. A success for tunnel B clears the flag and won't retry tunnel A —
	// but A's next pingcheck event will see alreadyFailed=false (after rollback)
	// and proceed normally, so self-healing works via the periodic event stream.
	dirty   bool
	stopCh  chan struct{}
	stopped chan struct{}

	// pending — последнее состояние pingcheck по туннелю, ещё не обработанное
	// (F287). Слушатель шины только кладёт сюда и будит исполнителя: сверка
	// идёт RCI-записями и может висеть, а пока слушатель в ней, буфер
	// подписчика (64) переполнялся и шина молча роняла события — в том числе
	// fail другого туннеля, который публикуется один раз на порог.
	// Схлопывание до последнего ЗНАЧИМОГО состояния (fail с failCount>0 или
	// pass) корректно: важен итог, а повтор после неудачной сверки (dirty)
	// держится на «следующем событии». Остаточный риск — всплеск быстрее
	// разбора: его шина по-прежнему роняет, но слушатель теперь только
	// разбирает и в RCI не ходит.
	pendingMu      sync.Mutex
	pending        map[string]events.PingCheckStateEvent
	wake           chan struct{}
	bus            *events.Bus
	log            Logger
	lookupAffected AffectedListsLookup
}

// NewFailoverManager creates a new failover manager.
// reconcileFn is called whenever the failed set changes; returns error to enable rollback.
// Pass nil if no reconcile callback is needed (for testing).
func NewFailoverManager(reconcileFn func() error) *FailoverManager {
	return &FailoverManager{
		failedSet:   make(map[string]struct{}),
		reconcileFn: reconcileFn,
	}
}

// SetEventBus sets the event bus for publishing failover notifications.
func (fm *FailoverManager) SetEventBus(bus *events.Bus) {
	fm.bus = bus
}

// SetLogger sets the logger for failover diagnostics.
func (fm *FailoverManager) SetLogger(log Logger) {
	fm.log = log
}

// SetAffectedListsLookup sets the callback that resolves which DNS lists are
// affected by a tunnel state change. Used to enrich SSE failover events.
func (fm *FailoverManager) SetAffectedListsLookup(lookup AffectedListsLookup) {
	fm.lookupAffected = lookup
}

// publishFailoverEvents publishes one SSE event per affected DNS list.
// If reconcileErr is non-nil, publishes "error" action with the error message.
// If lookupAffected is nil or returns no lists, no events are published.
func (fm *FailoverManager) publishFailoverEvents(tunnelID, baseAction string, reconcileErr error) {
	if fm.bus == nil || fm.lookupAffected == nil {
		return
	}

	affected := fm.lookupAffected(tunnelID, baseAction)
	if len(affected) == 0 {
		return
	}

	action := baseAction
	errMsg := ""
	if reconcileErr != nil {
		action = "error"
		errMsg = reconcileErr.Error()
	}

	for _, a := range affected {
		fm.bus.Publish("dnsroute:failover", events.DNSRouteFailoverEvent{
			ListID:     a.ListID,
			ListName:   a.ListName,
			TunnelID:   tunnelID,
			FromTunnel: a.FromTunnel,
			ToTunnel:   a.ToTunnel,
			Action:     action,
			Error:      errMsg,
		})
	}
}

// MarkFailed adds a tunnel to the failed set and triggers reconcile.
// If reconcile fails, the change is rolled back and the dirty flag is set
// so the next event will force-retry even if it's a duplicate.
// Returns nil on success, or the reconcile error.
func (fm *FailoverManager) MarkFailed(tunnelID string) error {
	fm.mu.Lock()
	wasDirty := fm.dirty
	_, alreadyFailed := fm.failedSet[tunnelID]

	// Skip no-op only if state is clean (no pending retry needed)
	if alreadyFailed && !wasDirty {
		fm.mu.Unlock()
		return nil
	}

	if !alreadyFailed {
		fm.failedSet[tunnelID] = struct{}{}
	}
	fm.mu.Unlock()

	if fm.reconcileFn == nil {
		return nil
	}

	err := fm.reconcileFn()

	fm.mu.Lock()
	if err != nil {
		// Rollback only if WE added it (not if it was already there before this call)
		if !alreadyFailed {
			delete(fm.failedSet, tunnelID)
		}
		fm.dirty = true
	} else {
		fm.dirty = false
	}
	fm.mu.Unlock()

	// Журналим только реальный переход (не dirty-retry от чужих событий)
	// и только когда у туннеля есть DNS-листы — иначе «переключение»
	// не происходило и строка вводит в заблуждение.
	if fm.log != nil && !alreadyFailed {
		if n := fm.affectedCount(tunnelID, "switched"); n != 0 {
			if err != nil {
				fm.log.Warnf("failover: DNS lists switch off failed tunnel %s did not apply: %v", tunnelID, err)
			} else if n > 0 {
				fm.log.Warnf("failover: tunnel %s failed, %d DNS list(s) switched to backup", tunnelID, n)
			} else {
				fm.log.Warnf("failover: tunnel %s failed, DNS lists switched to backup", tunnelID)
			}
		}
	}
	fm.publishFailoverEvents(tunnelID, "switched", err)
	return err
}

// affectedCount возвращает число DNS-листов, затронутых сменой состояния
// туннеля; -1 когда lookup не подключён (журналим без числа? нет —
// считаем «неизвестно» как «есть», чтобы не потерять событие).
func (fm *FailoverManager) affectedCount(tunnelID, action string) int {
	if fm.lookupAffected == nil {
		return -1
	}
	return len(fm.lookupAffected(tunnelID, action))
}

// MarkRecovered removes a tunnel from the failed set and triggers reconcile.
// If reconcile fails, the change is rolled back and the dirty flag is set.
// Returns nil on success, or the reconcile error.
func (fm *FailoverManager) MarkRecovered(tunnelID string) error {
	fm.mu.Lock()
	wasDirty := fm.dirty
	_, wasFailed := fm.failedSet[tunnelID]

	// Skip no-op only if state is clean
	if !wasFailed && !wasDirty {
		fm.mu.Unlock()
		return nil
	}

	if wasFailed {
		delete(fm.failedSet, tunnelID)
	}
	fm.mu.Unlock()

	if fm.reconcileFn == nil {
		return nil
	}

	err := fm.reconcileFn()

	fm.mu.Lock()
	if err != nil {
		if wasFailed {
			fm.failedSet[tunnelID] = struct{}{}
		}
		fm.dirty = true
	} else {
		fm.dirty = false
	}
	fm.mu.Unlock()

	if fm.log != nil && wasFailed {
		if n := fm.affectedCount(tunnelID, "restored"); n != 0 {
			if err != nil {
				fm.log.Warnf("failover: DNS lists restore to recovered tunnel %s did not apply: %v", tunnelID, err)
			} else {
				fm.log.Infof("failover: tunnel %s recovered, DNS lists restored", tunnelID)
			}
		}
	}
	fm.publishFailoverEvents(tunnelID, "restored", err)
	return err
}

// IsFailed checks if a tunnel is in the failed set.
func (fm *FailoverManager) IsFailed(tunnelID string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	_, exists := fm.failedSet[tunnelID]
	return exists
}

// FailedTunnels returns a copy of all currently failed tunnel IDs.
func (fm *FailoverManager) FailedTunnels() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	result := make([]string, 0, len(fm.failedSet))
	for id := range fm.failedSet {
		result = append(result, id)
	}
	return result
}

// StartListener subscribes to the event bus and listens for pingcheck:state events.
// Runs a goroutine that processes events until StopListener is called.
func (fm *FailoverManager) StartListener(bus *events.Bus) {
	fm.stopCh = make(chan struct{})
	fm.stopped = make(chan struct{})
	fm.pending = make(map[string]events.PingCheckStateEvent)
	fm.wake = make(chan struct{}, 1)

	_, ch, unsub := bus.Subscribe()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		wg.Wait()
		close(fm.stopped)
	}()

	// Слушатель: только разбор и постановка в pending — без RCI.
	go func() {
		defer wg.Done()
		defer unsub()
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ev.Type != "pingcheck:state" {
					continue
				}
				fm.enqueuePingCheckEvent(ev.Data)
			case <-fm.stopCh:
				return
			}
		}
	}()

	// Исполнитель: сверки по одной, в своём темпе.
	go func() {
		defer wg.Done()
		for {
			select {
			case <-fm.wake:
				for _, ev := range fm.takePending() {
					select {
					case <-fm.stopCh:
						return
					default:
					}
					fm.handlePingCheckEvent(ev)
				}
			case <-fm.stopCh:
				return
			}
		}
	}()
}

func (fm *FailoverManager) enqueuePingCheckEvent(data any) {
	ev, ok := data.(events.PingCheckStateEvent)
	if !ok {
		// Разбор типа и предупреждение — у handlePingCheckEvent.
		fm.handlePingCheckEvent(data)
		return
	}
	// В очередь — только то, на что исполнитель отреагирует: игнорируемое
	// событие (стартовый fail с failCount=0, прочие статусы) затёрло бы в
	// pending настоящее состояние туннеля, и переход потерялся бы.
	if !significantPingCheck(ev) {
		return
	}
	fm.pendingMu.Lock()
	fm.pending[ev.TunnelID] = ev
	fm.pendingMu.Unlock()
	select {
	case fm.wake <- struct{}{}:
	default: // исполнитель уже разбужен — заберёт и это
	}
}

func (fm *FailoverManager) takePending() []events.PingCheckStateEvent {
	fm.pendingMu.Lock()
	defer fm.pendingMu.Unlock()
	out := make([]events.PingCheckStateEvent, 0, len(fm.pending))
	for id, ev := range fm.pending {
		out = append(out, ev)
		delete(fm.pending, id)
	}
	return out
}

// StopListener stops the event listener goroutine.
func (fm *FailoverManager) StopListener() {
	if fm.stopCh != nil {
		close(fm.stopCh)
		<-fm.stopped
	}
}

func (fm *FailoverManager) handlePingCheckEvent(data any) {
	ev, ok := data.(events.PingCheckStateEvent)
	if !ok {
		if fm.log != nil {
			fm.log.Warnf("dnsroute failover: unexpected pingcheck event type: %T", data)
		}
		return
	}

	if !significantPingCheck(ev) {
		return
	}
	if ev.Status == "fail" {
		_ = fm.MarkFailed(ev.TunnelID)
	} else {
		_ = fm.MarkRecovered(ev.TunnelID)
	}
}

// significantPingCheck — событие, на которое failover реагирует: pass или
// настоящий fail. NDMS шлёт status="fail" с failCount=0 на старте туннеля, до
// первой проверки — это ещё не отказ; прочие статусы (пустой на прогреве NWG)
// failover не касаются.
func significantPingCheck(ev events.PingCheckStateEvent) bool {
	return ev.Status == "pass" || (ev.Status == "fail" && ev.FailCount > 0)
}
