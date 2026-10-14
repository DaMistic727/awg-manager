package singbox

import (
	"errors"
	"strings"
	"testing"
)

// MC8: FATAL sing-box цитирует адрес remote rule_set целиком; токен в пути и
// query не должен доезжать ни до lastError, ни до причины краша — их отдают
// статусы REST и MCP-ключа только для чтения.
func TestHandleExit_RedactsURLsInStatus(t *testing.T) {
	o := newOperatorForTest(t)
	tail := `FATAL[0000] start service: initialize rule-set[geo]: Get "https://rules.example.com/u/SECRETPATH/set.srs?token=SECRETQ": EOF`
	o.handleExit(errors.New("exit status 1"), tail, false)

	_, reason, _ := o.CrashStats()
	for name, got := range map[string]string{"LastError": o.LastError(), "crash reason": reason} {
		for _, secret := range []string{"SECRETPATH", "SECRETQ"} {
			if strings.Contains(got, secret) {
				t.Fatalf("%s несёт %s: %q", name, secret, got)
			}
		}
		if !strings.Contains(got, "initialize rule-set") {
			t.Fatalf("%s: суть ошибки потеряна: %q", name, got)
		}
	}
}
