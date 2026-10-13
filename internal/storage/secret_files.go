package storage

// DeviceKeyFile — имя файла секрета устройства в dataDir, которым шифровался
// ключ подписки Amnezia Premium. Писателя больше нет; миграция V41 файл
// стирает, а internal/backup по этому имени не пускает в архив то, что
// стереть не удалось.
const DeviceKeyFile = ".device-key"

// RCITokenFile — токен доступа к RCI (KeeneticOS 5.2+), см. ndms/transport/token.go.
// Привязан к выпустившему роутеру и даёт права admin: в бэкап не едет.
const RCITokenFile = "rci-token"
