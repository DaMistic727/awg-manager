package api

import (
	"archive/zip"
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/storage"
)

func TestExport_AppendsInstanceSection(t *testing.T) {
	h, store := newTunnelsUpdateHarness(t, &stubTunnelSvc{})
	seedObfTunnel(t, store)
	rec := httptest.NewRecorder()
	h.Export(rec, httptest.NewRequest(http.MethodGet, "/tunnels/export?id=awg20", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Endpoint = 127.0.0.1:39000") || !strings.Contains(body, "[instance]") || !strings.Contains(body, "target = 1.2.3.4:51824") {
		t.Fatalf("export:\n%s", body)
	}
}

func TestUpdate_RejectsInvalidObfuscator(t *testing.T) {
	h, store := newTunnelsUpdateHarness(t, &stubTunnelSvc{})
	seedObfTunnel(t, store)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPost, "/tunnels/update?id=awg20",
		strings.NewReader(`{"obfuscator":{"target":"1.2.3.4:1","key":"k","masking":"TLS"}}`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "INVALID_OBFUSCATOR") {
		t.Fatalf("expected 400 INVALID_OBFUSCATOR: %d %s", rec.Code, rec.Body.String())
	}
	if saved, _ := store.Get("awg20"); saved.Obfuscator.Masking != "STUN" {
		t.Fatal("invalid update must not persist")
	}
}

// F646: имя — свободная строка; заголовок обязан разбираться обратно в то
// же имя (кавычка, кириллица), а не обрываться на кавычке.
func TestExport_ContentDispositionRoundTrips(t *testing.T) {
	h, store := newTunnelsUpdateHarness(t, &stubTunnelSvc{})
	if err := store.Create(&storage.AWGTunnel{ID: "awg1", Name: `Гер"мания`,
		Interface: storage.AWGInterface{Address: "10.0.0.2/32", PrivateKey: "k"},
		Peer:      storage.AWGPeer{PublicKey: "p", Endpoint: "1.2.3.4:51820"}}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Export(rec, httptest.NewRequest(http.MethodGet, "/tunnels/export?id=awg1", nil))
	_, params, err := mime.ParseMediaType(rec.Header().Get("Content-Disposition"))
	if err != nil || params["filename"] != `Гер"мания.conf` {
		t.Fatalf("Content-Disposition %q → %v, %v", rec.Header().Get("Content-Disposition"), params, err)
	}
}

// F647: записи архива — без разделителей пути (zip-slip) и без дублей.
func TestExportAll_EntryNamesSafeAndDistinct(t *testing.T) {
	h, store := newTunnelsUpdateHarness(t, &stubTunnelSvc{})
	for id, name := range map[string]string{"awg1": "../../x", "awg2": "Германия", "awg3": "Германия"} {
		if err := store.Create(&storage.AWGTunnel{ID: id, Name: name,
			Interface: storage.AWGInterface{Address: "10.0.0.2/32", PrivateKey: "k"},
			Peer:      storage.AWGPeer{PublicKey: "p", Endpoint: "1.2.3.4:51820"}}); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	h.ExportAll(rec, httptest.NewRequest(http.MethodGet, "/tunnels/export-all", nil))
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("zip: %v (body %q)", err, rec.Body.String())
	}
	seen := map[string]bool{}
	for _, f := range zr.File {
		if strings.ContainsAny(f.Name, `/\`) || seen[f.Name] {
			t.Errorf("запись %q: разделитель пути или дубль", f.Name)
		}
		seen[f.Name] = true
	}
	if len(seen) != 3 {
		t.Fatalf("записей %d, want 3: %v", len(seen), seen)
	}
}

// F662: «Экспорт всех» несёт [instance] обфусцированного туннеля, как и
// одиночный экспорт.
func TestExportAll_AppendsInstanceSection(t *testing.T) {
	h, store := newTunnelsUpdateHarness(t, &stubTunnelSvc{})
	seedObfTunnel(t, store)
	rec := httptest.NewRecorder()
	h.ExportAll(rec, httptest.NewRequest(http.MethodGet, "/tunnels/export-all", nil))
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil || len(zr.File) != 1 {
		t.Fatalf("zip: %v files=%d", err, len(zr.File))
	}
	f, _ := zr.File[0].Open()
	body, _ := io.ReadAll(f)
	if !strings.Contains(string(body), "[instance]") || !strings.Contains(string(body), "target = 1.2.3.4:51824") {
		t.Fatalf("архив без [instance]:\n%s", body)
	}
}
