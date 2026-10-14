package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type fakeLifecycle struct {
	stopCalled  bool
	startCalled bool
	stopErr     error
	startErr    error
}

func (f *fakeLifecycle) Stop(ctx context.Context) error  { f.stopCalled = true; return f.stopErr }
func (f *fakeLifecycle) Start(ctx context.Context) error { f.startCalled = true; return f.startErr }

func newTestServer(t *testing.T, body []byte) (*httptest.Server, string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	sum := sha256.Sum256(body)
	return srv, hex.EncodeToString(sum[:])
}

func TestMigrate_HappyPath(t *testing.T) {
	body := []byte("new managed binary")
	srv, hexSum := newTestServer(t, body)

	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: srv.URL, SHA256: hexSum}, nil)
	inst.SetDownloader(&testHTTPDownloader{})
	inst.opkgRemove = func(ctx context.Context) error { return nil }

	lc := &fakeLifecycle{}
	if err := inst.Migrate(context.Background(), lc); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !lc.stopCalled {
		t.Error("Lifecycle.Stop not called")
	}
	if !lc.startCalled {
		t.Error("Lifecycle.Start not called")
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("managed binary not in place: %v", err)
	}
}

func TestMigrate_AbortsBeforeOpkgRemoveOnDownloadFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	// Network failure (port 1 — connection refused) makes Download fail
	// before any SHA verification. SHA mismatch is covered separately by
	// TestInstaller_Download_SHAMismatch in installer_test.go; both
	// failure modes must abort before opkg-remove.
	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: "http://127.0.0.1:1/never", SHA256: "0000"}, nil)

	opkgCalled := false
	inst.opkgRemove = func(ctx context.Context) error { opkgCalled = true; return nil }

	lc := &fakeLifecycle{}
	if err := inst.Migrate(context.Background(), lc); err == nil {
		t.Fatal("expected Migrate to fail, got nil")
	}
	if opkgCalled {
		t.Error("opkgRemove called even though download failed — must abort before destructive op")
	}
	if lc.stopCalled {
		t.Error("Lifecycle.Stop called even though download failed")
	}
}

func TestMigrate_IdempotentWhenManagedAlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	// Управляемый бинарь на месте — миграция не нужна.
	scriptBody := "#!/bin/sh\necho 'sing-box version 1.13.11'\n"
	if err := os.WriteFile(target, []byte(scriptBody), 0755); err != nil {
		t.Fatal(err)
	}

	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: "http://127.0.0.1:1/never", SHA256: "0000"}, nil)
	opkgCalled := false
	inst.opkgRemove = func(ctx context.Context) error { opkgCalled = true; return nil }

	lc := &fakeLifecycle{}
	if err := inst.Migrate(context.Background(), lc); err != nil {
		t.Fatalf("Migrate idempotent path failed: %v", err)
	}
	if opkgCalled || lc.stopCalled || lc.startCalled {
		t.Errorf("Migrate triggered side effects when managed binary already in place: opkg=%v stop=%v start=%v", opkgCalled, lc.stopCalled, lc.startCalled)
	}
}

// F134: бинарь на месте, но `version` молчит (UPX не распаковался при
// нехватке RAM) — это не «бинаря нет», миграция поверх него не идёт.
func TestMigrate_SkipsWhenBinaryPresentButVersionSilent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: "http://127.0.0.1:1/never", SHA256: "0000"}, nil)
	opkgCalled := false
	inst.opkgRemove = func(ctx context.Context) error { opkgCalled = true; return nil }

	lc := &fakeLifecycle{}
	if err := inst.Migrate(context.Background(), lc); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if opkgCalled || lc.stopCalled {
		t.Errorf("миграция поверх живого бинаря: opkg=%v stop=%v", opkgCalled, lc.stopCalled)
	}
}

func TestMigrate_OpkgRemoveErrorIsLogged_NotFatal(t *testing.T) {
	body := []byte("new managed binary")
	srv, hexSum := newTestServer(t, body)

	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: srv.URL, SHA256: hexSum}, nil)
	inst.SetDownloader(&testHTTPDownloader{})
	// opkg returns an error (e.g. lock contention) — Migrate should still
	// activate the managed binary and start the daemon.
	inst.opkgRemove = func(ctx context.Context) error { return os.ErrInvalid }

	lc := &fakeLifecycle{}
	if err := inst.Migrate(context.Background(), lc); err != nil {
		t.Fatalf("Migrate should tolerate opkg failure: %v", err)
	}
	if !lc.startCalled {
		t.Error("Lifecycle.Start not called after opkg failure")
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("managed binary not in place after opkg failure: %v", err)
	}
}

func TestIsLegacyOpkgInstalled_ParsesEntwareOutput(t *testing.T) {
	// Smoke-test that the function returns false on a system where opkg is
	// not present (test runner) — we don't have a way to mock the exec call
	// without a bigger refactor. The negative path is the important one.
	dir := t.TempDir()
	inst := New(filepath.Join(dir, "sb"), "test", BinarySpec{}, nil)
	if inst.IsLegacyOpkgInstalled(context.Background()) {
		t.Error("IsLegacyOpkgInstalled should return false on a system without opkg")
	}
}

func TestMigrate_TolerantOfStopFailure(t *testing.T) {
	body := []byte("new managed binary")
	srv, hexSum := newTestServer(t, body)

	dir := t.TempDir()
	target := filepath.Join(dir, "sing-box")
	inst := New(target, "test", BinarySpec{Version: "1.13.11", URL: srv.URL, SHA256: hexSum}, nil)
	inst.SetDownloader(&testHTTPDownloader{})
	inst.opkgRemove = func(ctx context.Context) error { return nil }

	// Stop returns an error — Migrate should log and continue, NOT abort.
	lc := &fakeLifecycle{stopErr: os.ErrClosed}
	if err := inst.Migrate(context.Background(), lc); err != nil {
		t.Fatalf("Migrate should tolerate Stop failure: %v", err)
	}
	if !lc.startCalled {
		t.Error("Lifecycle.Start should still be called even when Stop failed")
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("managed binary not in place: %v", err)
	}
}

func TestIsLegacyOpkgInstalled_PositivePath(t *testing.T) {
	dir := t.TempDir()
	inst := New(filepath.Join(dir, "sb"), "test", BinarySpec{}, nil)
	inst.opkgListInstalled = func(ctx context.Context) (string, error) {
		// Mimic Entware `opkg list-installed` output format.
		return "ca-certificates - 20240203-2\nsing-box-naive - 1.13.8-1\nawg-manager - 2.9.10-1\n", nil
	}
	if !inst.IsLegacyOpkgInstalled(context.Background()) {
		t.Error("IsLegacyOpkgInstalled should return true when sing-box-naive is in opkg list")
	}

	// Negative: package absent.
	inst.opkgListInstalled = func(ctx context.Context) (string, error) {
		return "ca-certificates - 20240203-2\nawg-manager - 2.9.10-1\n", nil
	}
	if inst.IsLegacyOpkgInstalled(context.Background()) {
		t.Error("IsLegacyOpkgInstalled should return false when sing-box-naive absent")
	}

	// Negative: opkg returns error.
	inst.opkgListInstalled = func(ctx context.Context) (string, error) {
		return "", os.ErrNotExist
	}
	if inst.IsLegacyOpkgInstalled(context.Background()) {
		t.Error("IsLegacyOpkgInstalled should return false on opkg error")
	}
}
