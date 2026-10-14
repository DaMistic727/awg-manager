package singbox

import "github.com/hoaxisr/awg-manager/internal/logging"

func sanitizeSingboxLogText(s string) string {
	// Runtime masking is shared with /api/logs DTO projection in
	// internal/logging. URL сначала режется до схемы и хоста (MC8): FATAL
	// sing-box цитирует адрес remote rule_set целиком, с токеном в пути или
	// query, а отсюда текст уходит в lastError и причину краша — статусы REST
	// и MCP-ключа только для чтения. Порядок тот же, что у журнала (RedactURLs,
	// затем SanitizeLogText). Текст выхода процесса проходит обёртку дважды
	// (process.go маскирует хвост, handleExit — ещё раз): маска хоста уже без
	// точки, и URL в статусе сворачивается в scheme://<redacted> — подсказка
	// о хосте там теряется, токен не выживает в обоих случаях.
	return logging.SanitizeLogText(logging.RedactURLs(s))
}
