# Запуск проекта

Перед запуском убедитесь, что у вас установлен Docker и Docker Compose.

Поднимите инфраструктуру через file service

Накатите миграции

`bash
go run cmd/migrate/main.go -cmd=up
`
Запустите сервис 

`bash
go run cmd/servers
`

# Миграции

В проекте используется инструмент golang-migrate для управления схемой базы данных.

Миграции не выполняются автоматически при старте приложения.  
Они запускаются вручную через отдельный Go‑бинарь cmd/migrate.

Это позволяет:

- контролировать порядок изменений,
- безопасно откатывать миграции,
- легко встроить миграции в CI/CD.

---

# Установка CLI (опционально)

CLI может пригодиться для локальной разработки:

`bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
`

Проверка:

`bash
migrate -version
`

---

# Структура миграций

Все SQL‑файлы находятся в:

`
internal/db/migrations/
`

Каждая миграция состоит из двух файлов:

- NNNNN_name.up.sql — изменения «вперёд»  
- NNNNN_name.down.sql — откат изменений

Пример:

`
000001_init.up.sql
000001_init.down.sql
`

---

# Создание новой миграции

`bash
migrate create -ext sql -dir internal/db/migrations -seq {{migration_name}}
`

Будут созданы файлы:

`
internal/db/migrations/000002migrationname.up.sql
internal/db/migrations/000002migrationname.down.sql
`

---

# Применение миграций

1. Применить все миграции

`bash
go run ./cmd/migrate -cmd up
`

2. Откатить одну миграцию

`bash
go run ./cmd/migrate -cmd down -steps 1
`

3. Принудительно выставить версию

`bash
go run ./cmd/migrate -cmd force -version 1
`

---

# Основные команды

| Команда | Описание |
|--------|----------|
| up | применить все новые миграции |
| down | откатить указанное количество миграций |
| force N | принудительно выставить версию N |
| version | показать текущую версию схемы |

---

# Обновление Protobuf

# 1. Установка protobuf-compiler (уже сделано)
sudo apt install -y protobuf-compiler

# 2. Установка Go плагинов
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 3. Проверка PATH
echo $PATH | tr ':' '\n' | grep go

# 4. Проверка установки плагинов
protoc-gen-go --version
protoc-gen-go-grpc --version

# 5. Запуск make
make gen-proto
