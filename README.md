# EstimatePro

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white)
![License](https://img.shields.io/badge/license-AGPL--3.0-blue)
![Version](https://img.shields.io/badge/version-0.12.8-blue)

**Коллаборативная платформа для оценки проектов.**

Загружайте ТЗ в любом формате, собирайте оценки команды и получайте автоматическую агрегацию по методологии PERT. Вся история версий и оценок — в одном месте.

## Основные возможности

* **Командная оценка** — PM создаёт проект, приглашает разработчиков, каждый загружает свою оценку независимо
* **PERT-агрегация** — сводная таблица с avg, min, max по каждой задаче. Расхождения оценок видны мгновенно
* **Загрузка документов** — ТЗ в PDF, DOCX, XLSX, Markdown, TXT, CSV. Версионирование файлов в MinIO/S3
* **Гибкие роли** — Admin, PM, Tech Lead, Developer, Observer. Каждая роль видит только то, что нужно
* **OAuth2** — вход через Google и GitHub с синхронизацией аватара и имени
* **Real-time обновления** — WebSocket уведомления с toast-оповещениями (Sonner)
* **Уведомления** — in-app, email и Telegram. Настраиваемые предпочтения по каналам
* **Мультиязычность** — Русский (по умолчанию) и English. Уведомления на языке пользователя
* **Тёмная/светлая тема** — Адаптивный интерфейс с Three.js Aurora шейдером на лендинге
* **JWT авторизация** — Access/Refresh токены в Redis, авто-обновление при 401
* **UI/UX концепт** — Дизайн-макеты всех экранов в `frontend/design/concept.pen` (light + dark)

## Архитектура

Построено на принципах **модульного монолита** с Clean Architecture:

```
backend/                         frontend/
├── cmd/server/main.go           ├── app/[locale]/
├── internal/                    │   ├── page.tsx (Landing)
│   ├── modules/                 │   ├── (auth)/ (Login/Register)
│   │   ├── auth/                │   ├── auth/callback/ (OAuth)
│   │   ├── project/             │   └── dashboard/
│   │   ├── document/            │       ├── page.tsx (Workspace)
│   │   ├── estimation/          │       ├── projects/
│   │   ├── notify/              │       ├── notifications/
│   │   └── ws/                  │       └── settings/
│   ├── shared/                  ├── components/ui/
│   │   ├── middleware/          ├── features/
│   │   ├── errors/              │   ├── auth/
│   │   ├── pagination/          │   ├── projects/
│   │   └── response/            │   ├── documents/
│   └── infra/                   │   ├── estimation/
│       ├── postgres/            │   ├── notifications/
│       ├── redis/               │   └── activity/
│       └── s3/                  └── lib/
└── pkg/jwt/
```

**Правила:**
- Domain → Usecase → Handler (dependency rule)
- Нет cross-module imports (взаимодействие через интерфейсы)
- Нет бизнес-логики в handler-ах
- Table-driven тесты рядом с source

## Быстрый старт

### Системные требования

- **Go** 1.26+
- **Node.js** 24+
- **PostgreSQL** 18
- **Redis** 8
- **Docker** & Docker Compose (OrbStack рекомендуется)

### Запуск

```bash
# Инфраструктура (postgres, redis, minio, backend)
just dev-infra

# Backend (альтернатива — без Docker)
cd backend && export $(grep -v '^#' ../.env | xargs) && go run ./cmd/server

# Frontend
cd frontend && npm install && npm run dev

# Миграции
just migrate-up

# Тесты backend
just test-backend
# или
cd backend && go test ./...
```

### Переменные окружения

Скопируйте `.env.example` → `.env` и заполните:

| Переменная | Описание | По умолчанию |
|-----------|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:password@localhost:5432/estimatepro` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` |
| `JWT_SECRET` | Секрет для JWT токенов | — |
| `JWT_ACCESS_TTL` | Время жизни access токена | `15m` |
| `JWT_REFRESH_TTL` | Время жизни refresh токена | `168h` (7 дней) |
| `S3_ENDPOINT` | MinIO/S3 endpoint | `localhost:9000` |
| `S3_ACCESS_KEY` | S3 access key | `minioadmin` |
| `S3_SECRET_KEY` | S3 secret key | `minioadmin` |
| `S3_BUCKET` | S3 bucket name | `documents` |
| `SERVER_PORT` | Порт backend сервера | `8080` |

## Технологический стек

### Backend

| Технология | Назначение |
|-----------|-----------|
| Go 1.26 | Язык backend |
| Chi | HTTP router |
| pgx | PostgreSQL драйвер |
| go-redis | Redis клиент |
| MinIO SDK | S3-совместимое хранилище |
| gorilla/websocket | WebSocket real-time |
| oauth2 | OAuth2 (Google, GitHub) |
| bcrypt | Хэширование паролей |
| JWT (HMAC) | Аутентификация |

### Frontend

| Технология | Назначение |
|-----------|-----------|
| Next.js 16 | React фреймворк (App Router) |
| TypeScript 5 | Типизация |
| Tailwind CSS v4 | Стилизация |
| TanStack Query | Серверный state |
| Zustand | UI state |
| next-intl | Интернационализация (ru/en) |
| shadcn/ui | UI компоненты |
| Framer Motion | Анимации |
| Sonner | Toast-уведомления |
| Three.js | Aurora шейдер на лендинге |

### Инфраструктура

| Технология | Назначение |
|-----------|-----------|
| PostgreSQL 18 | Основная БД |
| Redis 8 | Кэш, refresh токены |
| MinIO | Хранилище документов (S3) |
| Docker Compose | Оркестрация |
| GitHub Actions | CI/CD |

## API Endpoints

### Auth
| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/auth/register` | Регистрация |
| POST | `/api/v1/auth/login` | Вход |
| POST | `/api/v1/auth/refresh` | Обновление токена |
| GET | `/api/v1/auth/me` | Текущий пользователь |
| PATCH | `/api/v1/auth/profile` | Обновление профиля |

### OAuth
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/auth/oauth/:provider` | Начало OAuth flow (google/github) |
| GET | `/api/v1/auth/oauth/:provider/callback` | Callback OAuth |

### Notifications
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/notifications` | Список уведомлений |
| GET | `/api/v1/notifications/unread-count` | Кол-во непрочитанных |
| PATCH | `/api/v1/notifications/read-all` | Отметить все прочитанными |
| PATCH | `/api/v1/notifications/:id/read` | Отметить прочитанным |
| GET | `/api/v1/notifications/preferences` | Настройки уведомлений |
| PUT | `/api/v1/notifications/preferences` | Обновить настройки |

### WebSocket
| Путь | Описание |
|------|----------|
| `/api/v1/ws` | WebSocket подключение (JWT auth через query param) |

### Workspaces
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/workspaces` | Список пространств |
| POST | `/api/v1/workspaces` | Создание пространства |

### Projects
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/projects` | Список проектов (по workspace или по user) |
| POST | `/api/v1/projects` | Создание проекта |
| GET | `/api/v1/projects/:id` | Получение проекта |
| PATCH | `/api/v1/projects/:id` | Обновление проекта |
| DELETE | `/api/v1/projects/:id` | Архивация проекта |
| POST | `/api/v1/projects/:id/restore` | Восстановление |

### Members
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/projects/:id/members` | Список участников |
| POST | `/api/v1/projects/:id/members` | Добавление участника |
| DELETE | `/api/v1/projects/:id/members/:userId` | Удаление участника |

### Documents
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/projects/:id/documents` | Список документов |
| POST | `/api/v1/projects/:id/documents` | Загрузка (multipart) |
| GET | `/api/v1/projects/:id/documents/:docId` | Получение |
| GET | `/api/v1/projects/:id/documents/:docId/download` | Скачивание |
| DELETE | `/api/v1/projects/:id/documents/:docId` | Удаление |

### Estimations
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/projects/:id/estimations` | Список оценок (`?mine=true`) |
| POST | `/api/v1/projects/:id/estimations` | Создание оценки |
| GET | `/api/v1/projects/:id/estimations/:estId` | Получение с items |
| PUT | `/api/v1/projects/:id/estimations/:estId/submit` | Отправка оценки |
| DELETE | `/api/v1/projects/:id/estimations/:estId` | Удаление черновика |
| GET | `/api/v1/projects/:id/estimations/aggregated` | PERT-агрегация |

## Роли

| Роль | Управление участниками | Создание оценок | Просмотр сводки |
|------|----------------------|----------------|----------------|
| Admin | Да | Да | Да |
| PM | Да | Да | Да |
| Tech Lead | Нет | Да | Да |
| Developer | Нет | Да | Да |
| Observer | Нет | Нет | Да |

## Тестирование

```bash
# Backend — 106 тестов
cd backend && go test ./... -v

# Линтинг
go vet ./...

# Frontend — TypeScript проверка
cd frontend && npx tsc --noEmit
```

### Покрытие тестами (backend)

| Модуль | Тесты | Покрытие |
|--------|-------|---------|
| `pkg/jwt` | 7 | Генерация, валидация, истечение |
| `auth/usecase` | 9 | Register, Login, Refresh, OAuth |
| `project/domain` | 3 | CanManageMembers, IsValid, CanEstimate |
| `project/usecase` | 14 | CRUD, Members, ListByUser |
| `project/handler` | 6 | Workspace handlers |
| `document/domain` | 2 | FileType.IsValid, MaxFileSize |
| `document/usecase` | 7 | Upload, List, Get, Delete |
| `estimation/domain` | 9 | PERT, Aggregation, Status |
| `estimation/parser` | 12 | Парсинг документов |
| `estimation/usecase` | 10 | CRUD, Submit, Aggregate |
| `notify/usecase` | 13 | List, MarkRead, Preferences, Dispatch |
| `notify/channel` | 4 | Email, Telegram каналы |
| `ws` | 3 | Hub, Broadcast, Subscribe |
| `shared/middleware` | 2 | JWT Auth, UserIDFromContext |
| `shared/pagination` | 2 | Parse, Offset |

## Версионирование

Проект следует [Semantic Versioning](https://semver.org/):

**Текущая версия: `0.12.8`**

### Changelog

#### v0.12.8 (2026-05-04)
- fix(bot/llm): formatter теперь явно проверяет HTTP status code от провайдеров (Claude/OpenAI/Grok/Ollama). 401/429/5xx больше не маскируются как «empty response» — `slog.WarnContext` показывает status + body preview (200 байт), error содержит `status %d`. Format() upper-level fallback на raw actionResult сохранён, поведение для пользователя не меняется (#41).
- fix(bot/llm): `io.ReadAll(resp.Body)` error больше не игнорируется через `_ =` — partial-read network failure (например, connection reset) теперь surface'ится с реальной причиной, а не как json unmarshal error на truncated buffer (#41).

#### v0.12.7 (2026-05-04)
- chore(bot): закрыты blind spots в логировании. `bot/repository/postgres.go` — 15 error-path'ей теперь дополнены `slog.ErrorContext` (Session/UserLink/LLMConfig/Memory/UserPrefs CRUD); следуют существующему in-file pattern (Memory.Save и UserLink.GetByTelegramUserID уже логировали так же). `bot/usecase/session.go:Advance` — добавлен log на `GetState` unmarshal failure. Поведение не меняется, только наблюдаемость.

#### v0.12.6 (2026-05-03)
- feat(bot/notify): intent `request_estimation` теперь реально шлёт уведомления участникам проекта через notify dispatcher вместо «функция в разработке». Добавлен `EventEstimationRequested` (`notify/domain`), новый sync-метод `Dispatcher.RequestEstimation(ctx, projectID, userID, taskName)`, `botEstimationAdapter` в композиционном корне форвардит вызов (#24).
- refactor(notify): `eventMeta` map стал единым источником шаблонов для async (`HandleEvent`) и sync (`RequestEstimation`) путей. Новый флаг `SyncOnly` в записи делает явным, что некоторые события (сейчас — `estimation.requested`) маршрутизируются только через типизированные методы — generic async path их отвергает и не создаёт notification с пустым taskName.
- chore(bot/domain): `ErrFeatureNotImplemented` удалён вместе с placeholder-кодом — sentinel жил только пока adapter не был подключён, по ADR-014 без consumer'а в domain ему не место.

#### v0.12.5 (2026-05-03)
- fix(bot/usecase): добавление участника снова работает end-to-end. После клика по кнопке роли в `add_member` сессия теперь автоматически выполняется (раньше зависала до 10-минутного TTL — нет шага Confirm). `executeSessionAction` для AddMember/RemoveMember резолвит `project_name → project_id` через `findProjectByName` (и `user_name → user_id` через новый `findMemberByName`) — ранее читался пустой `state["project_id"]`, AddByEmail/Remove падали на UUID-валидации (#27).
- fix(bot/usecase): ошибки `ErrProjectNotFound` и нового `ErrMemberNotFound` маппятся в named-сообщения (`Проект «X» не найден.`, `Участник «X» не найден.`) вместо обезличенного `Ошибка...` — `sessionActionErrorMessage` helper держит контекст консистентным с intent.go Execute-flow.
- chore(bot/domain): новый sentinel `ErrMemberNotFound` для resolve user_name по списку участников (симметрично `ErrProjectNotFound`).

#### v0.12.4 (2026-05-03)
- fix(bot/usecase): `ProcessCallback` отвергает callback'и с unknown `sel_<key>` — раньше parser-side принимал любой `sel_*` префикс и пушил его как произвольное поле в session state. Producer-side helpers (`SelectCallback`/`SelectAction`) уже паникуют на unknown `CallbackKey`, теперь parser симметричен: warn-log + `AnswerCallbackQuery` + early return (#35).

#### v0.12.3 (2026-05-02)
- refactor(bot): typed `CallbackAction` + `ParseCallback` для type-safe парсера. ProcessCallback теперь идёт через `action.IsCancel()`/`IsConfirm()`/`IsSelect()`/`SelectKey()`, никаких `strings.HasPrefix`/`TrimPrefix` для callback-протокола. `SelectAction` возвращает `CallbackAction` (#31, #32).
- fix(bot/domain): `ConfirmCallback` реджектит `IntentUnknown` — classifier эмитит его на unparseable input, и подтверждать там нечего (#33).
- chore(bot): private `parseCallbackData` helper удалён из usecase, поднят в `bot/domain/callback.go` как `ParseCallback`. Single source of truth для wire-format split.

#### v0.12.2 (2026-05-02)
- refactor(bot): callback-протокол вынесен в typed constants `bot/domain/callback.go` — magic strings `cancel`/`confirm:<intent>`/`sel_<key>:<value>` заменены на `CancelCallback()`/`ConfirmCallback(intent)`/`SelectCallback(key, value)`/`SelectAction(key)` (#29)
- feat(bot/domain): typed `CallbackKey` (whitelist `proj`/`role` через `IsKnown()`) + конструкторы panic-ят на invalid input — programmer-error инвариант идиомой `regexp.MustCompile`. Wire-format не изменился, легасные inline-keyboards в чатах продолжают работать.

#### v0.12.1 (2026-05-02)
- audit(bot): callback completion flow audit (#21) — найдены 3 бага в multi-step flows
- fix(bot): добавление участника снова работает — role-кнопки теперь отдают `sel_role:*` callback вместо `role:*`, который silent-ignored в ProcessCallback (#26)
- chore: AddMember/RemoveMember executeSessionAction state-key mismatch + missing auto-execute зафиксированы как #27 для следующего PR

#### v0.12.0 (2026-05-02)
- feat(bot): реализованы все 4 ранее неработавших intents — classifier их распознавал, executor падал в default → unknown (#19)
  - `update_project` — переименование/изменение описания через session-flow с подтверждением
  - `submit_estimation` — отправка PERT-оценки (min/likely/max) в одну команду
  - `request_estimation` — запрос оценки от команды (placeholder, real notify-integration → #24)
  - `upload_document` (text-flow) — text-intent создаёт сессию, следующий файл загружается в правильный проект автоматически
- feat(bot): защитный gate `TestExecute_AllValidIntentsHaveCase` — валидные intents без case в Execute теперь fail тестов
- feat(bot): sentinel errors `ErrInvalidEstimationHours` и `ErrFeatureNotImplemented` — никакого silent-success в адаптерах
- chore(bot): classifier prompt расширен `new_name` параметром для update_project + два примера

#### v0.11.4 (2026-05-02)
- fix(bot): команды «участники [проект]» и «оценка [проект]» из help снова работают по тексту — `listMembers`/`getAggregated` теперь резолвят project по имени через общий `findProjectByName` helper, а не падают на пустом `project_id` (#18)
- refactor(bot): sentinel errors `ErrProjectNotIdentified`/`ErrProjectNotFound` в `bot/domain`, `getProjectStatus` мигрирован на тот же helper, `projectListLimit` константа

#### v0.11.3 (2026-05-02)
- fix(bot): унифицирован формат `callback_data` для кнопок «Отмена» — везде `"cancel:"` (action:payload convention). Парсер `parseCallbackData` сохраняет backward-compat для legacy `"cancel"` без двоеточия (#20)

#### v0.11.2 (2026-04-23)
- Retry транзиентных ошибок Telegram API (5xx, 429, TLS timeout) — до 3 попыток с exponential backoff
- REACTION_INVALID silenced — ожидаемая ситуация в чатах с ограниченными реакциями

#### v0.11.1 (2026-04-23)
- Конфигурируемый LOG_LEVEL из ENV (debug/info/warn/error)

#### v0.11.0 (2026-04-22)
- Автолинковка Telegram-пользователей: бот автоматически привязывает аккаунт при первом сообщении по telegram_chat_id из профиля
- Comprehensive slog logging по всему модулю бота (142 вызова в 13 файлах): handler, usecase, LLM-провайдеры, telegram client, repository

#### v0.10.0 (2026-04-17)
- DDD-конструкторы для всех 13 domain entities с централизованной валидацией инвариантов
- Архитектурные гейты: `arch-check.sh` (5 правил), CI gate, PR template с обязательным code-review
- EmailRateLimiter/RateLimit с context-based graceful shutdown
- WorkspaceUsecase: handler thin, бизнес-логика в usecase
- Typed MemoryRole enum, ProfileUpdate с pointer-семантикой
- Test coverage 70%+ (backend + frontend), testcontainers-go
- Bugfixes: CreateProject 500→400, goroutine leaks, data race в dispatcher

#### v0.7.0 (2026-03-25)
- **UI/UX Design Concept**: `frontend/design/concept.pen` — 12 экранов (6 страниц × light/dark)
  - Landing, Login, Register, Dashboard, Project Detail, Settings
  - Разработка UI/UX велась по концепту, максимально близко к реализации
- Dashboard: estimation chart (вертикальные бары по проектам), stats, workspaces, project progress
- Settings: Profile (аватар + имя), Appearance (ThemeToggle), Notifications (3 канала с Switch)
- Обновлена проектная документация

#### v0.6.0 (2026-03-20)
- Document version flags: чекбокс «Подписана» и «Финальная версия»
- Хэштеги для версий документов (max 3, 9 предустановленных + кастомные)
- WebSocket real-time обновления с Sonner toast и анимированным сердечком
- OAuth2 интеграция Google и GitHub (полный flow + sync avatar/name)
- Аватар профиля: загрузка в MinIO, отображение через JWT, кеширование

#### v0.4.0 (2026-03-19)
- Redis refresh tokens с ротацией и logout endpoint
- User avatar с upload и membership-based access control
- Hover card профиля в header dashboard
- Code quality: writeJSON deduplicated, cross-module import fix, RoleChecker refactor
- 86 backend тестов, удалено 2000+ строк мёртвого кода

#### v0.3.0 (2026-03-19)
- Модуль оценок (Estimation): CRUD, PERT-агрегация, сводная таблица
- Динамический dashboard: workspace карточки, графики, прогресс проектов
- Timeline проекта с градиентными цветами и реальными данными
- Activity logs (стиль 21st.dev) на странице уведомлений
- Code quality: writeJSON deduplicated, cross-module import fix, RoleChecker refactor
- 77 backend тестов, удалено 2000+ строк мёртвого кода

#### v0.2.0 (2026-03-17)
- Auth module: JWT авторизация, register/login/refresh
- Project module: CRUD, members, roles
- Document module: upload/download/delete в MinIO
- Frontend: Landing, Dashboard, Project detail с Timeline

#### v0.1.0 (2026-03-16)
- Инициализация проекта
- Инфраструктура: Docker Compose, PostgreSQL, Redis, MinIO
- CI/CD: GitHub Actions

## Лицензия

Copyright © 2026 Daniil Vdovin.

Этот проект лицензирован под **GNU Affero General Public License v3.0** — см. файл [LICENSE](LICENSE).

Если вы модифицируете код и предоставляете его как сетевой сервис, вы обязаны опубликовать исходный код ваших изменений под той же лицензией.

Для коммерческого использования без ограничений AGPL — свяжитесь с автором для получения коммерческой лицензии.

### Контрибьюции

Мы приветствуем вклад в проект! Перед отправкой Pull Request ознакомьтесь с [CONTRIBUTING.md](CONTRIBUTING.md) и подпишите [CLA](CLA.md).
