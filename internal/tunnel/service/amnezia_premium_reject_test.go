package service

import (
	"context"
	"errors"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/storage"
	"github.com/hoaxisr/awg-manager/internal/tunnel/config"
)

// premiumConf — форма .conf с портала Amnezia Premium: ключ подписки в
// заголовке. Значение синтетическое.
const premiumConf = "# VPN Key: vpn://AAAA\n" + sampleConf

// Импорт и замена — оба пути, которыми .conf попадает в запись туннеля
// (web и MCP), — отвергают конфигурацию подписки, не тронув хранилище.
func TestImportAndReplace_RejectAmneziaPremium(t *testing.T) {
	s, _, _ := serviceForImport(t)
	if _, err := s.Import(context.Background(), premiumConf, "p", "kernel", ImportLink{}); !errors.Is(err, config.ErrAmneziaPremium) {
		t.Fatalf("Import: err = %v, want ErrAmneziaPremium", err)
	}
	if list, _ := s.store.List(); len(list) != 0 {
		t.Fatalf("запись заведена при отказе: %d", len(list))
	}

	s, _ = serviceWithStore(t)
	if err := s.store.Create(&storage.AWGTunnel{ID: "awg10", Name: "t", Backend: "kernel"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceConfig(context.Background(), "awg10", premiumConf, ""); !errors.Is(err, config.ErrAmneziaPremium) {
		t.Fatalf("ReplaceConfig: err = %v, want ErrAmneziaPremium", err)
	}
	if got, _ := s.store.Get("awg10"); got.Peer.PublicKey != "" {
		t.Fatalf("конфигурация заменена при отказе: %+v", got.Peer)
	}
}
