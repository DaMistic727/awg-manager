package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// V41 снимает с флеша всё, что осталось от Amnezia Premium: шифротекст
// ключа подписки, адрес зеркала и страну подключения. Ключ обязан пропасть
// и из .bak — туда прежний файл попадает жёсткой ссылкой при первой же
// записи, и без перезаписи копия жила бы рядом с .device-key.
func TestMigrateToV41_DropsAmneziaPremium(t *testing.T) {
	const cipher = "fixture-premium-cipher-0123456789"
	dir := t.TempDir()
	seed := `{
  "schemaVersion": 40,
  "amneziaPremiumKeyCipher": "` + cipher + `",
  "amneziaPremiumMirrorUrl": "https://mirror.test/cp",
  "amneziaPremiumDeclaredCountry": "ru"
}`
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("запись настроек: %v", err)
	}

	s, err := NewSettingsStore(dir).Load()
	if err != nil {
		t.Fatalf("загрузка: %v", err)
	}
	if s.SchemaVersion != 41 {
		t.Fatalf("schemaVersion = %d, want 41", s.SchemaVersion)
	}
	for _, f := range []string{path, path + ".bak"} {
		got := readFile(t, f)
		for _, gone := range []string{cipher, "amneziaPremium"} {
			if strings.Contains(got, gone) {
				t.Errorf("%s после миграции содержит %q", filepath.Base(f), gone)
			}
		}
	}
}

// Основной файл испорчен, настройки поднимаются из .bak v40 с шифротекстом:
// прежнего основного файла нет, но .bak переписаться обязан так же.
// settings.json.corrupt — карантинная копия, её не разбирают и не трогают;
// шифротекст в ней мёртв, потому что секрет устройства стёрт (тест ниже).
func TestMigrateToV41_RestoreFromBakDropsCipher(t *testing.T) {
	const cipher = "fixture-premium-cipher-bak-0123456789"
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{испорчен"), 0o600); err != nil {
		t.Fatal(err)
	}
	bak := `{"schemaVersion": 40, "amneziaPremiumKeyCipher": "` + cipher + `"}`
	if err := os.WriteFile(path+".bak", []byte(bak), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewSettingsStore(dir).Load(); err != nil {
		t.Fatalf("загрузка: %v", err)
	}
	for _, f := range []string{path, path + ".bak"} {
		if strings.Contains(readFile(t, f), cipher) {
			t.Errorf("%s после восстановления содержит шифротекст", filepath.Base(f))
		}
	}
}

// Секрет устройства и его временные/карантинные копии стираются; соседние
// файлы каталога данных — нет.
func TestMigrateToV41_RemovesDeviceKey(t *testing.T) {
	dir := t.TempDir()
	gone := []string{DeviceKeyFile, DeviceKeyFile + ".corrupt", DeviceKeyFile + ".new.123"}
	kept := []string{RCITokenFile, "device-key-notes"}
	for _, n := range append(append([]string{}, gone...), kept...) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"schemaVersion": 40}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewSettingsStore(dir).Load(); err != nil {
		t.Fatalf("загрузка: %v", err)
	}
	for _, n := range gone {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Errorf("%s не стёрт: %v", n, err)
		}
	}
	for _, n := range kept {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s задет: %v", n, err)
		}
	}
}
