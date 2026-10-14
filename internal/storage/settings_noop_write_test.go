package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F645: мутатор, ничего не поменявший, не переписывает файл (износ флеша) —
// атомарная запись меняет inode, поэтому тот же inode = записи не было.
// Настоящая правка и файл с правами старой версии записываются.
func TestSettingsUpdate_NoChangeSkipsWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewSettingsStore(dir)
	if _, err := store.Load(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	noop := func(*Settings) error { return nil }
	if err := store.Update(noop); err != nil { // первая запись: файла могло не быть
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Update(noop); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.Stat(path); !os.SameFile(before, after) {
		t.Fatal("холостой Update переписал файл")
	}

	on := true
	if err := store.Update(func(s *Settings) error { s.McpEnabled = on; return nil }); err != nil {
		t.Fatal(err)
	}
	changed, _ := os.Stat(path)
	if os.SameFile(before, changed) {
		t.Fatal("настоящая правка не записана")
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(noop); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != SecretFilePermission {
		t.Fatalf("права старого файла не починены: %v", fi.Mode().Perm())
	}
}

// F645 не отменяет F254: если в .bak остался снятый секрет (перезапись .bak
// когда-то сорвалась), холостое сохранение обязано его вылечить, а не
// пропуститься.
func TestSettingsUpdate_NoChangeStillHealsStaleBak(t *testing.T) {
	dir := t.TempDir()
	store := NewSettingsStore(dir)
	if _, err := store.Load(); err != nil {
		t.Fatal(err)
	}
	noop := func(*Settings) error { return nil }
	if err := store.Update(noop); err != nil {
		t.Fatal(err)
	}
	bak := filepath.Join(dir, "settings.json.bak")
	if err := os.WriteFile(bak, []byte(`{"apiKey":"STALE-SECRET"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(noop); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(bak); strings.Contains(string(b), "STALE-SECRET") {
		t.Fatal("холостое сохранение оставило снятый секрет в .bak")
	}
}
