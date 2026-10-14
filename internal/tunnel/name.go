package tunnel

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxNameBytes — предел имени туннеля в байтах UTF-8. Имя — это описание
// записи интерфейса в NDMS, а NDMS принимает описание не длиннее 256 байт:
// на 257 он заводит запись с ПУСТЫМ описанием и отвечает ошибкой (стенд,
// 2026-09-28). Своя запись с пустым описанием для F517 чужая — туннель не
// стартовал бы никогда.
const MaxNameBytes = 256

// ErrNameTooLong — имя не влезает в описание записи NDMS.
var ErrNameTooLong = fmt.Errorf("имя туннеля длиннее %d байт (ограничение роутера)", MaxNameBytes)

// ErrNameEmpty — имя пустое или из одних пробелов. Имя — описание записи
// OpkgTun, по которому туннель узнаёт свою запись (F517); пустое описание
// старт и остановка трактовали по-разному (F531).
var ErrNameEmpty = errors.New("имя туннеля не может быть пустым")

// ValidateName — отказ на пустом имени и на имени длиннее MaxNameBytes.
// Байты, а не руны: кириллица занимает по два байта, предел — 128 букв.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNameEmpty
	}
	if len(name) > MaxNameBytes {
		return fmt.Errorf("%w: %d байт", ErrNameTooLong, len(name))
	}
	return nil
}

// TruncateName — для имени, которое подставил не человек (из ссылки): оно
// обрезается до MaxNameBytes по границе руны, а не отвергается.
func TruncateName(name string) string {
	if len(name) <= MaxNameBytes {
		return name
	}
	cut := MaxNameBytes
	for cut > 0 && !utf8.RuneStart(name[cut]) {
		cut--
	}
	return name[:cut]
}
