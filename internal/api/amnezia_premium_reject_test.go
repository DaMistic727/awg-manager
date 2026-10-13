package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/storage"
	"github.com/hoaxisr/awg-manager/internal/tunnel"
	"github.com/hoaxisr/awg-manager/internal/tunnel/config"
)

// premiumAddrConf — конфигурация подписки без заголовка портала: опознаётся
// по адресу интерфейса.
const premiumAddrConf = "[Interface]\nPrivateKey = K==\nAddress = 100.102.228.17/32\n[Peer]\nPublicKey = P==\nEndpoint = 192.0.2.1:51820\n"

// Пользователь видит текст отказа дословно — без префикса «parse conf:».
func wantPremiumRefusal(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("тело не JSON: %s", rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest || body.Message != config.ErrAmneziaPremium.Error() || body.Code != "AMNEZIA_PREMIUM_UNSUPPORTED" {
		t.Fatalf("код = %d, тело: %s", rec.Code, rec.Body.String())
	}
}

func TestImportConf_AmneziaPremiumRefused(t *testing.T) {
	svc := &importStubSvc{}
	h := NewImportHandler(svc, storage.NewAWGTunnelStore(t.TempDir()), &appLogSpy{})
	body, _ := json.Marshal(map[string]string{"content": premiumAddrConf, "name": "p"})
	rec := httptest.NewRecorder()
	h.ImportConf(rec, httptest.NewRequest(http.MethodPost, "/api/import/conf", bytes.NewReader(body)))

	wantPremiumRefusal(t, rec)
	if svc.content != "" {
		t.Fatal("конфигурация дошла до Import")
	}
}

// Отказ — ДО Stop: иначе работающий туннель остался бы выключенным.
func TestTunnelReplaceConf_AmneziaPremiumRefusedBeforeStop(t *testing.T) {
	stopped := 0
	stub := &stubTunnelSvc{
		stateFn: func(string) tunnel.StateInfo { return tunnel.StateInfo{State: tunnel.StateRunning} },
		stopFn:  func(context.Context, string) error { stopped++; return nil },
	}
	h, store := newTunnelsUpdateHarness(t, stub)
	if err := store.Create(&storage.AWGTunnel{ID: "awg11", Name: "NL"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"content": "# VPN Key: vpn://AAAA\n" + premiumAddrConf})
	rec := httptest.NewRecorder()
	h.ReplaceConf(rec, httptest.NewRequest(http.MethodPost, "/tunnels/replace?id=awg11", bytes.NewReader(body)))

	wantPremiumRefusal(t, rec)
	if stopped != 0 || stub.replaceCalls != 0 {
		t.Fatalf("туннель тронут: stop=%d replace=%d", stopped, stub.replaceCalls)
	}
}
