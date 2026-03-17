с# EstimatePro — Project Foundation

> Этот документ является основой проекта. Новая сессия Claude Code должна прочитать его целиком перед началом разработки и следовать всем решениям, описанным здесь. Не предлагать альтернативы без явного запроса.

---

## 1. Концепция продукта

**EstimatePro** — коллаборативная платформа для оценки проектов.

### Проблема
Команды разработки оценивают проекты в разных форматах (Excel, PDF, MD, DOCX), обмениваются файлами через мессенджеры, теряют версии и не имеют единого места для агрегации оценок.

### Решение
Платформа где:
- PM создаёт проект (workspace) и добавляет участников
- PM загружает ТЗ в любом формате — система его парсит
- Разработчики получают уведомление (email / Telegram), скачивают ТЗ, загружают свою оценку
- Платформа агрегирует оценки всех участников в единую сводную таблицу
- Вся история загрузок и версий зафиксирована в одном месте

### Целевая аудитория (сейчас)
Собственная команда разработки. С заделом на SaaS-продажу студиям и фрилансерам.

### Монетизация (будущее)
SaaS — подписка за workspace/команду.

---

## 2. Роли пользователей

Фиксированные роли (не кастомные на MVP):

| Роль | Права |
|---|---|
| `Admin` | Полный доступ ко всему: управление workspace, пользователями, проектами, настройками платформы. Может назначать и снимать любые роли включая других Admin |
| `PM` | Создаёт проекты, загружает ТЗ, приглашает участников, видит агрегацию |
| `Tech Lead` | Загружает оценки, видит оценки других, управляет разработчиками |
| `Developer` | Загружает свою оценку, видит только свою и агрегат |
| `Observer` | Только чтение, без загрузки |

> Admin — системная роль уровня workspace. Первый пользователь создавший workspace автоматически становится Admin.

---

## 3. Ключевые сущности (Domain)

```
Workspace      — организация/команда
  └─ Project   — конкретный проект для оценки
       ├─ Member[]      — участники с ролями
       ├─ Document[]    — файлы ТЗ (от PM)
       │    └─ Version[] — версии документа
       └─ Estimation[]  — оценки участников
            └─ EstimationItem[] — строки оценки (задача → часы)
```

### Value Objects
- `Role` — Admin | PM | TechLead | Developer | Observer
- `FileType` — pdf | docx | xlsx | md | txt | csv
- `DocumentStatus` — pending | parsed | failed | needs_review
- `EstimationStatus` — draft | submitted | confirmed

### Domain Events
- `DocumentUploaded` — триггер для уведомлений
- `EstimationSubmitted` — триггер для пересчёта агрегата
- `MemberInvited` — триггер для welcome email
- `ProjectStatusChanged`

---

## 4. Бэкенд архитектура

### Стек
- **Go** — основной язык
- **Chi** — HTTP роутер
- **sqlc** + **pgx** — работа с PostgreSQL (типобезопасный SQL)
- **MinIO** — S3-совместимое хранилище файлов (self-hosted, легко мигрировать на AWS S3)
- **Redis** — сессии, кэш, очередь уведомлений
- **PostgreSQL** — основная БД

### Архитектурный подход
**Modular Monolith + Clean Architecture внутри каждого модуля + DDD-inspired паттерны + прагматичный TDD**

Что это означает на практике:
- Физически — один бинарник, один деплой
- Логически — изолированные модули по доменам, между ними только через интерфейсы или доменные события
- Dependency Rule: зависимости идут только внутрь (infra → usecase → domain), domain ничего не знает об infra
- DDD паттерны: Value Objects, Domain Events, Repository interfaces — но БЕЗ полного ритуала (нет Aggregate Roots, нет bounded context ceremony)
- TDD обязательно для: `estimation` модуля (парсинг + агрегация), `domain` слоя. Опционально для handlers на старте

### Структура папок (Go)

```
/cmd/
  server/
    main.go              ← точка входа, wire всё вместе

/internal/
  modules/
    project/             ← домен "проект"
      domain/
        project.go       ← entity Project
        interfaces.go    ← Repository interface (порт)
        events.go        ← доменные события
        errors.go        ← domain errors
      usecase/
        create_project.go
        invite_member.go
        archive_project.go
      handler/
        http.go          ← один файл на фичу (Vertical Slice внутри)
        dto.go
      repo/
        postgres.go      ← реализует domain.Repository

    document/            ← домен "документы"
      domain/
      usecase/
      handler/
      repo/

    estimation/          ← САМЫЙ ВАЖНЫЙ модуль, TDD обязательно
      domain/
      usecase/
      handler/
      repo/
      parser/            ← парсеры форматов (см. раздел 5)

    auth/
      domain/
      usecase/
      handler/

    notify/              ← уведомления
      email/             ← gomail + SMTP / Resend
      telegram/          ← go-telegram-bot-api

  shared/
    middleware/          ← auth JWT middleware, logging, CORS
    errors/              ← общие HTTP ошибки
    pagination/

  infra/
    postgres/            ← миграции, connection pool
    s3/                  ← MinIO client
    redis/               ← go-redis

/pkg/
  jwt/                   ← JWT утилиты
  validator/             ← валидация DTO

/migrations/             ← SQL миграции (goose)
```

### Что НЕ делать (антипаттерны)
- ❌ Не импортировать один модуль напрямую в другой (только через события или интерфейсы)
- ❌ Не класть бизнес-логику в handlers
- ❌ Не использовать глобальные переменные
- ❌ Полный DDD overhead (агрегаты с инвариантами, фабрики, ubiquitous language ceremony)

---

## 5. Document Processing Pipeline

Ключевая техническая задача — работа с любыми форматами файлов.

### Поддерживаемые форматы
`pdf`, `docx`, `xlsx`, `md`, `txt`, `csv`

### Пайплайн

```
Входящий файл
    ↓
Format Detector      ← MIME type + расширение
    ↓
Parser Router        ← выбирает нужный парсер
    ↓
Парсер:
  - pdfcpu / pdftotext  ← PDF
  - unioffice            ← DOCX
  - excelize             ← XLSX
  - goldmark             ← MD
  - stdlib               ← TXT, CSV
    ↓
ParsedDocument       ← единая внутренняя структура
  { title, sections[], tables[], rawText }
    ↓
Estimation Extractor ← ищет часы, задачи, суммы
    ↓
  confidence_score:
    1.0  → автоматически сохраняем EstimationItems
    <0.7 → помечаем needs_review, просим подтвердить
    ↓
Сохранение:
  - S3: raw blob + HTML preview
  - PostgreSQL: EstimationItems (если confidence достаточный)
```

### Confidence Score
Если экстрактор не уверен в извлечённых данных — помечает `needs_review`, НЕ блокирует процесс. Пользователь получает уведомление и может отредактировать вручную.

---

## 6. Агрегация оценок

После того как все разработчики загрузили оценки — PM видит сводную таблицу:

```
Задача              | Daniil  | Alex  | Maria | Avg  | Min  | Max
--------------------|---------|-------|-------|------|------|----
Auth module         |   8h    |  10h  |   8h  |  8.7h|  8h  | 10h
File upload service |  16h    |  20h  |  14h  | 16.7h| 14h  | 20h
...
ИТОГО               |  80h    |  95h  |  75h  | 83.3h| 75h  | 95h
```

SQL запрос агрегации — `SUM + GROUP BY estimation_items + JOIN members`.

---

## 7. Уведомления

**Архитектура**: `Notification Service` с событийной очередью через Redis.

**Каналы**:
- Email — `gomail` + SMTP (Resend или Mailgun на старте)
- Telegram — `go-telegram-bot-api`

**События для уведомлений**:
- `document.uploaded` → все разработчики проекта
- `estimation.submitted` → PM + Tech Lead
- `member.invited` → приглашённый пользователь
- `project.deadline_approaching` → все участники

**Расширяемость**: добавить новый канал (Slack, WhatsApp) = новый handler в очереди, бизнес-логика не меняется.

---

## 8. Фронтенд архитектура

### Стек
- **Next.js 14** (App Router)
- **TypeScript**
- **shadcn/ui** — основная UI библиотека (копируется в проект, не зависимость)
- **Tailwind CSS v4**
- **framer-motion** — анимации
- **lucide-react** — иконки
- **three.js** — только для лендинга (lazy import)
- **TanStack Query** — server state, мутации, invalidation
- **Zustand** — только UI state (модальные окна, фильтры, sidebar)

### UI компоненты (21st.dev)
Все устанавливаются через `npx shadcn@latest add <url>`:

```bash
npx shadcn@latest add https://21st.dev/r/moumensoliman/interactive-logs-table-shadcnui
npx shadcn@latest add https://21st.dev/r/aceternity/timeline
npx shadcn@latest add https://21st.dev/r/aceternity/sidebar
npx shadcn@latest add https://21st.dev/r/ayushmxxn/theme-toggle
npx shadcn@latest add https://21st.dev/r/minhxthanh/simple-ui
npx shadcn@latest add https://21st.dev/r/shadcnblockscom/hero-195-1
npx shadcn@latest add https://21st.dev/r/easemize/starfall-portfolio-landing
```

**Важно**: `starfall-portfolio-landing` использует `three.js` — подключать только через динамический импорт:
```tsx
const StarfallLanding = dynamic(
  () => import('@/components/ui/starfall-portfolio-landing'),
  { ssr: false }
)
```

### Структура папок (Next.js)

```
/app
  (landing)/
    page.tsx             ← Hero195 + Starfall (публичный лендинг)
    layout.tsx           ← без sidebar, без auth
  (auth)/
    login/page.tsx
    register/page.tsx
  (dashboard)/
    layout.tsx           ← Sidebar + auth middleware check
    page.tsx             ← список проектов
    projects/
      [id]/
        page.tsx         ← детали проекта
        documents/page.tsx
        estimation/page.tsx
        members/page.tsx
    settings/page.tsx
  api/                   ← Next.js API routes (только для auth callbacks)

/features                ← вертикальные слайсы по доменам
  projects/
    components/
      ProjectCard.tsx
      ProjectList.tsx
      CreateProjectModal.tsx
    hooks/
      useProjects.ts     ← TanStack Query hooks
      useCreateProject.ts
    api/
      projects.ts        ← типизированные fetch-функции
    types.ts
  documents/
    components/
      FileUpload.tsx      ← drag&drop загрузка
      DocumentVersion.tsx
      ParsedPreview.tsx
    hooks/
    api/
  estimation/
    components/
      EstimationForm.tsx
      AggregationTable.tsx  ← главная таблица агрегации
      ConfidenceWarning.tsx ← если score < 0.7
    hooks/
    api/
  members/
  notifications/

/components
  ui/                    ← shadcn/ui (автогенерируется)
  sidebar.tsx            ← Aceternity sidebar
  timeline.tsx           ← Aceternity timeline (история версий)
  logs-table.tsx         ← Interactive logs (audit log)
  theme-toggle.tsx

/lib
  api-client.ts          ← базовый fetch с auth headers + error handling
  query-client.ts        ← TanStack Query конфиг
  auth.ts                ← JWT utils на клиенте

/store
  ui.ts                  ← Zustand: sidebar open/closed, modal states, filters

/types
  api.ts                 ← типы из OpenAPI (генерируются через openapi-typescript)
```

### Правила работы с данными

| Что | Как | Почему |
|---|---|---|
| Список проектов, детали | Server Component + fetch | 0 JS в бандле, SEO |
| Upload файла, submit оценки | TanStack Query mutation | Оптимистичный апдейт, retry |
| Агрегационная таблица | TanStack Query + polling | Обновляется когда кто-то загружает |
| Sidebar open/closed | Zustand | Чистый UI state |
| Модальное окно | Zustand | Чистый UI state |
| Серверные данные | НИКОГДА в Zustand | |

---

## 9. База данных (PostgreSQL)

### Ключевые таблицы

```sql
-- Пространства
workspaces (id, name, owner_id, created_at)
workspace_members (workspace_id, user_id, role, invited_at, joined_at)

-- Проекты
projects (id, workspace_id, name, description, status, created_by, created_at)
project_members (project_id, user_id, role)

-- Документы (ТЗ от PM)
documents (id, project_id, title, uploaded_by, created_at)
document_versions (
  id, document_id, version_number,
  file_key,          -- путь в S3
  file_type,         -- pdf|docx|xlsx|md|txt|csv
  file_size,
  parsed_status,     -- pending|parsed|failed|needs_review
  confidence_score,  -- 0.0–1.0
  uploaded_by,
  uploaded_at
)

-- Оценки разработчиков
estimations (
  id, project_id, document_version_id,
  submitted_by, status,  -- draft|submitted|confirmed
  submitted_at
)
estimation_items (
  id, estimation_id,
  task_name, hours,
  sort_order
)

-- Уведомления
notification_preferences (user_id, channel, enabled)
notification_log (id, user_id, event_type, channel, sent_at, status)
```

### Миграции
Используем `goose`. Файлы в `/migrations/`, нумерация `001_init.sql`, `002_add_notifications.sql` и т.д.

---

## 10. API Design

### Соглашения
- REST JSON
- Базовый URL: `/api/v1/`
- Авторизация: `Authorization: Bearer <jwt>`
- Ошибки: `{ "error": { "code": "NOT_FOUND", "message": "..." } }`
- Пагинация: `?page=1&limit=20` → `{ "data": [], "meta": { "total", "page", "limit" } }`

### Основные эндпоинты

```
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh

GET    /api/v1/workspaces
POST   /api/v1/workspaces

GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/:id
PATCH  /api/v1/projects/:id
DELETE /api/v1/projects/:id

POST   /api/v1/projects/:id/members
DELETE /api/v1/projects/:id/members/:userId

GET    /api/v1/projects/:id/documents
POST   /api/v1/projects/:id/documents          ← multipart upload
GET    /api/v1/projects/:id/documents/:docId/versions
POST   /api/v1/projects/:id/documents/:docId/versions  ← новая версия

GET    /api/v1/projects/:id/estimations
POST   /api/v1/projects/:id/estimations        ← submit оценки
GET    /api/v1/projects/:id/estimations/aggregate  ← сводная таблица

GET    /api/v1/notifications/preferences
PATCH  /api/v1/notifications/preferences
```

### Типизация
Бэкенд генерирует OpenAPI spec через `swaggo/swag`.
Фронтенд генерирует TypeScript типы через `openapi-typescript`:
```bash
npx openapi-typescript http://localhost:8080/swagger/doc.json -o types/api.ts
```

---

## 11. Зависимости (npm install)

```bash
# Основные
npm install framer-motion lucide-react three @tanstack/react-query zustand

# Dev
npm install -D @types/three openapi-typescript

# shadcn/ui инициализация (один раз)
npx shadcn@latest init
```

---

## 12. Зависимости (Go — go.mod)

```
github.com/go-chi/chi/v5
github.com/jackc/pgx/v5
github.com/redis/go-redis/v9
github.com/golang-jwt/jwt/v5
github.com/swaggo/swag
github.com/pdfcpu/pdfcpu
github.com/xuri/excelize/v2
github.com/yuin/goldmark
github.com/unidoc/unioffice          ← DOCX парсинг
gopkg.in/mail.v2                      ← gomail для email
github.com/go-telegram-bot-api/telegram-bot-api/v5
github.com/pressly/goose/v3          ← миграции
github.com/testcontainers/testcontainers-go  ← интеграционные тесты
```

---

## 13. TDD стратегия

### Обязательно (TDD с первого дня)
- `estimation/parser/` — каждый формат, граничные случаи, кириллица, пустые файлы
- `estimation/domain/` — агрегационная логика, confidence score
- `project/domain/` — бизнес-правила (нельзя удалить проект с активными оценками и т.д.)

### Рекомендуется
- `document/domain/` — версионирование, валидация форматов
- Use case слой каждого модуля (с моками репозиториев)

### Интеграционные (testcontainers-go)
- Репозитории — реальная БД в Docker контейнере
- API handlers — e2e через HTTP

### Не нужно на старте
- 100% coverage handlers
- Тесты UI компонентов (Playwright — потом)

---

## 14. Окружение и запуск

### Docker Compose (dev)

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: estimatepro
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    ports: ["5432:5432"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"   ← S3 API
      - "9001:9001"   ← Web UI
```

### Переменные окружения (Go)

```env
DATABASE_URL=postgres://user:password@localhost:5432/estimatepro
REDIS_URL=redis://localhost:6379
S3_ENDPOINT=localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=estimatepro
JWT_SECRET=your-secret-key
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USER=resend
SMTP_PASSWORD=your-resend-api-key
TELEGRAM_BOT_TOKEN=your-bot-token
```

---

## 15. Многоязычность (i18n)

### Языки
- `ru` — русский (основной, дефолтный)
- `en` — английский

### Стек
**Next.js фронтенд**: `next-intl` — лучшая интеграция с App Router, поддержка Server Components, типобезопасные переводы.

```bash
npm install next-intl
```

### Структура переводов

```
/messages
  ru.json        ← русские строки (дефолт)
  en.json        ← английские строки
```

Пример структуры файла переводов:
```json
{
  "common": {
    "save": "Сохранить",
    "cancel": "Отмена",
    "loading": "Загрузка...",
    "error": "Ошибка"
  },
  "auth": {
    "login": "Войти",
    "logout": "Выйти",
    "email": "Email",
    "password": "Пароль"
  },
  "projects": {
    "title": "Проекты",
    "create": "Создать проект",
    "empty": "Нет проектов"
  },
  "estimation": {
    "submit": "Отправить оценку",
    "aggregate": "Сводная таблица",
    "hours": "часов"
  },
  "roles": {
    "admin": "Администратор",
    "pm": "Менеджер проекта",
    "tech_lead": "Тех. лид",
    "developer": "Разработчик",
    "observer": "Наблюдатель"
  }
}
```

### Конфигурация next-intl

```ts
// i18n.ts
import { getRequestConfig } from 'next-intl/server'

export default getRequestConfig(async ({ locale }) => ({
  messages: (await import(`./messages/${locale}.json`)).default
}))
```

```ts
// middleware.ts
import createMiddleware from 'next-intl/middleware'

export default createMiddleware({
  locales: ['ru', 'en'],
  defaultLocale: 'ru',
  localePrefix: 'as-needed'  // /en/... для английского, / для русского
})

export const config = {
  matcher: ['/((?!api|_next|.*\\..*).*)']
}
```

### Роутинг с локалями

```
/                  ← ru (дефолт, без префикса)
/en                ← en
/projects          ← ru
/en/projects       ← en
```

### Использование в компонентах

```tsx
// Server Component
import { useTranslations } from 'next-intl'

export default function ProjectsPage() {
  const t = useTranslations('projects')
  return <h1>{t('title')}</h1>
}

// Client Component
'use client'
import { useTranslations } from 'next-intl'

export function CreateButton() {
  const t = useTranslations('projects')
  return <button>{t('create')}</button>
}
```

### Переключатель языка
Компонент в `components/locale-switcher.tsx` — простой dropdown с двумя опциями. Сохраняет выбор в cookie через next-intl middleware.

### Go бэкенд (i18n для уведомлений и ошибок)
- Язык определяется из профиля пользователя (`preferred_locale` в таблице `users`)
- Email-шаблоны: отдельные файлы `templates/email/ru/` и `templates/email/en/`
- API ошибки возвращаются на английском (коды) + человекочитаемое сообщение на языке пользователя опционально

### БД изменения
```sql
-- добавить в таблицу users
ALTER TABLE users ADD COLUMN preferred_locale VARCHAR(5) DEFAULT 'ru';
```

### Приоритет реализации
i18n подключается на **Этапе 1** вместе с базовой структурой Next.js — значительно дешевле добавить сразу чем рефакторить позже. Переводы заполняются параллельно с разработкой фич.

---

## 16. Порядок разработки (MVP)

### Этап 1 — Основа (2 недели)
1. Настройка репозитория, Docker Compose, CI (GitHub Actions)
2. Go: `auth` модуль (JWT), базовые middleware
3. Go: `project` модуль (CRUD проектов, участники + роль Admin)
4. Next.js: auth pages, dashboard layout с Aceternity sidebar
5. Next.js: **next-intl подключить сразу**, базовые переводы ru/en
6. БД миграции: workspaces, projects, members (включая `preferred_locale`)

### Этап 2 — Документы (2 недели)
1. Go: `document` модуль — upload, версионирование, S3 интеграция
2. Go: Document Processing Pipeline — парсеры (TDD)
3. Next.js: FileUpload компонент, список версий с Timeline
4. Next.js: Preview загруженного документа

### Этап 3 — Оценки (2 недели)
1. Go: `estimation` модуль — submit оценки (TDD)
2. Go: Estimation Extractor с confidence score (TDD)
3. Go: Агрегационный запрос
4. Next.js: EstimationForm, AggregationTable
5. Next.js: Interactive Logs Table для audit log

### Этап 4 — Уведомления + полировка (1 неделя)
1. Go: `notify` модуль — email + Telegram (шаблоны на ru/en)
2. Next.js: настройки уведомлений, locale switcher
3. Лендинг: Hero195 + Starfall (динамический импорт three.js)
4. Theme toggle, финальная полировка

---

## 17. Что намеренно оставлено на потом

- Микросервисы — монолит сначала, разбивать когда будет реальная нагрузка
- CQRS / Event Sourcing — overkill для текущего масштаба
- Полный DDD (агрегаты, bounded contexts) — паттерны используем, ритуал нет
- WebSocket real-time — polling достаточно на MVP
- Kubernetes — Docker Compose сначала
- Playwright e2e тесты — после MVP
- Мобильное приложение — потом
- Кастомные роли — после выхода на рынок
- Дополнительные языки (помимо ru/en) — после первых продаж

---

## 18. Соглашения по коду

### Go
- Именование: `camelCase` для переменных, `PascalCase` для экспортируемых
- Ошибки: всегда оборачивать контекстом `fmt.Errorf("projectUseCase.Create: %w", err)`
- Интерфейсы: определяются в `domain/`, реализуются в `repo/` или `infra/`
- Тесты: `*_test.go` рядом с файлом, table-driven tests

### TypeScript / Next.js
- Компоненты: PascalCase, один файл = один компонент
- Хуки: `use` префикс, в `hooks/` папке фичи
- API функции: в `api/` папке фичи, типизированы через `types/api.ts`
- Никаких `any` — строгий TypeScript
- `'use client'` только там где реально нужна интерактивность
- Все строки через `useTranslations()` — никаких хардкодных строк в JSX

### Git
- Ветки: `feat/`, `fix/`, `chore/` префиксы
- Коммиты: conventional commits (`feat: add file upload`, `fix: parser crash on empty pdf`)
- PR на `main` только через review

---

*Документ создан по итогам архитектурного обсуждения. Версия 1.1 — добавлены роль Admin и i18n (ru/en).*
