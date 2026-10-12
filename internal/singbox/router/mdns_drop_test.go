package router

import (
	"encoding/json"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/singbox/orchestrator"
)

// #1037 (F636): мультикаст (в т.ч. mDNS) не должен доходить до hijack-dns — иначе
// сервер type local шлёт мультикаст в t2s*, тот возвращается через SOCKS и
// снова перехватывается. Правило живёт только в конфиге на диске.
func TestMaterializeConfig_MDNSDropFirstAndRestoredAway(t *testing.T) {
	m := ruleSetMaterializer{configDir: t.TempDir(), binary: "/opt/bin/sing-box"}
	user := Rule{DomainSuffix: []string{"example.com"}, Outbound: "direct"}
	cfg := &RouterConfig{Route: Route{Final: "direct", Rules: []Rule{
		{Type: "logical", Mode: "or", Rules: []Rule{{Protocol: "dns"}, {Port: []int{53}}}, Action: "hijack-dns"},
		user,
	}}}

	first, err := m.materializeConfig(orchestrator.SlotRouter, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Повторная материализация (без restore) не должна плодить копии.
	twice, err := m.materializeConfig(orchestrator.SlotRouter, first)
	if err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]*RouterConfig{"first": first, "twice": twice} {
		if len(out.Route.Rules) != 3 {
			t.Fatalf("%s: %d правил, ждали 3: %+v", name, len(out.Route.Rules), out.Route.Rules)
		}
		got, _ := json.Marshal(out.Route.Rules[0])
		if string(got) != `{"ip_cidr":["224.0.0.0/4","ff00::/8"],"action":"reject"}` {
			t.Errorf("%s: rules[0] = %s, ждали отбой мультикаста перед hijack-dns", name, got)
		}
		if out.Route.Rules[1].Action != "hijack-dns" {
			t.Errorf("%s: rules[1] = %+v, ждали hijack-dns", name, out.Route.Rules[1])
		}
	}

	wantJSON, _ := json.Marshal(cfg.Route.Rules)
	restored := m.restoreConfig(twice)
	if gotJSON, _ := json.Marshal(restored.Route.Rules); string(gotJSON) != string(wantJSON) {
		t.Errorf("restore должен вернуть пользовательский список:\nполучили %s\nждали    %s", gotJSON, wantJSON)
	}
	// Сырые загрузчики (applied-конфиг для EnsureSystemRules) тоже не должны его видеть.
	data, _ := json.Marshal(twice)
	parsed, err := parseRouterConfigBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if gotJSON, _ := json.Marshal(parsed.Route.Rules); string(gotJSON) != string(wantJSON) {
		t.Errorf("разбор файла должен снять правило:\nполучили %s\nждали    %s", gotJSON, wantJSON)
	}
}
