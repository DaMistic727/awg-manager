package signature

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hoaxisr/awg-manager/internal/ndms"
)

// HeaderOrDefault — значение H-параметра для .conf. Пустым его писать НЕЛЬЗЯ:
// `awg setconf` на строке `H1 = ` отвергает ВЕСЬ файл (`Line unrecognized:
// 'H1='` + `Configuration parsing error`, rc=1 — стенд 08.09, F167), то есть
// туннель не стартует вовсе. Пустое поле означает «как в ванильном
// WireGuard», а там H1..H4 — это номера типов сообщений 1..4: ровно то, что
// показывает `awg showconf` на интерфейсе без header-параметров.
func HeaderOrDefault(v string, msgType int) string {
	if strings.TrimSpace(v) == "" {
		return strconv.Itoa(msgType)
	}
	return v
}

// WriteASCConf пишет хвост [Interface] клиентского .conf: числовые и
// header-параметры — из снимка ASC интерфейса (raw), сигнатуру I1–I5 — из
// записи ПИРА (CONTEXT.md «Сигнатура AWG»: у сервера своей нет).
//
// Сервер без ASC (Jc == 0) — обычный WireGuard: ни числовых параметров, ни
// сигнатуры в конфиг не пишем, иначе клиент получит обфускацию, которой на
// сервере нет. Один writer на оба пути — managed-сервер (internal/managed)
// и системный (internal/api).
func WriteASCConf(b *strings.Builder, raw json.RawMessage, packets GeneratedPackets) {
	var ext ndms.ASCParamsExtended
	if err := json.Unmarshal(raw, &ext); err != nil || ext.Jc == 0 {
		return
	}

	fmt.Fprintf(b, "Jc = %d\n", ext.Jc)
	fmt.Fprintf(b, "Jmin = %d\n", ext.Jmin)
	fmt.Fprintf(b, "Jmax = %d\n", ext.Jmax)
	fmt.Fprintf(b, "S1 = %d\n", ext.S1)
	fmt.Fprintf(b, "S2 = %d\n", ext.S2)
	fmt.Fprintf(b, "H1 = %s\n", HeaderOrDefault(ext.H1, 1))
	fmt.Fprintf(b, "H2 = %s\n", HeaderOrDefault(ext.H2, 2))
	fmt.Fprintf(b, "H3 = %s\n", HeaderOrDefault(ext.H3, 3))
	fmt.Fprintf(b, "H4 = %s\n", HeaderOrDefault(ext.H4, 4))

	if ext.S3 > 0 || ext.S4 > 0 {
		fmt.Fprintf(b, "S3 = %d\n", ext.S3)
		fmt.Fprintf(b, "S4 = %d\n", ext.S4)
	}
	for i, sig := range []string{packets.I1, packets.I2, packets.I3, packets.I4, packets.I5} {
		if sig != "" {
			fmt.Fprintf(b, "I%d = %s\n", i+1, sig)
		}
	}
}
