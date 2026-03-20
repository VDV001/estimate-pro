# EstimatePro

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Version](https://img.shields.io/badge/version-0.5.0-blue)

**Коллаборативная платформа для оценки проектов.**

Загружайте ТЗ в любом формате, собирайте оценки команды и получайте автоматическую агрегацию по методологии PERT. Вся история версий и оценок — в одном месте.

## Основные возможности

* **Командная оценка** — PM создаёт проект, приглашает разработчиков, каждый загружает свою оценку независимо
* **PERT-агрегация** — сводная таблица с avg, min, max по каждой задаче. Расхождения оценок видны мгновенно
* **Загрузка документов** — ТЗ в PDF, DOCX, XLSX, Markdown, TXT, CSV. Версионирование файлов в MinIO/S3
* **Гибкие роли** — Admin, PM, Tech Lead, Developer, Observer. Каждая роль видит только то, что нужно
* **Мультиязычность** — Русский (по умолчанию) и English. Уведомления на языке пользователя
* **Тёмная/светлая тема** — Адаптивный интерфейс с Three.js Aurora шейдером на лендинге
* **JWT авторизация** — Access/Refresh токены, авто-обновление при 401

## Архитектура

Построено на принципах **модульного монолита** с Clean Architecture:

```
backend/                         frontend/
├── cmd/server/main.go           ├── app/[locale]/
├── internal/                    │   ├── page.tsx (Landing)
│   ├── modules/                 │   ├── (auth)/ (Login/Register)
│   │   ├── auth/                │   └── dashboard/
│   │   ├── project/             │       ├── page.tsx (Workspace)
│   │   ├── document/            │       ├── projects/
│   │   └── estimation/          │       ├── notifications/
│   ├── shared/                  │       └── settings/
│   │   ├── middleware/          ├── components/ui/
│   │   ├── errors/              ├── features/
│   │   ├── pagination/          │   ├── auth/
│   │   └── response/            │   ├── projects/
│   └── infra/                   │   ├── documents/
│       ├── postgres/            │   ├── estimation/
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
# Backend — 77 тестов
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
| `auth/usecase` | 7 | Register, Login, Refresh |
| `project/domain` | 17 | CanManageMembers, IsValid, CanEstimate |
| `project/usecase` | 14 | CRUD, Members, ListByUser |
| `project/handler` | 6 | Workspace handlers |
| `document/domain` | 10 | FileType.IsValid, MaxFileSize |
| `document/usecase` | 11 | Upload, List, Get, Delete |
| `estimation/domain` | 16 | PERT, Aggregation, Status |
| `estimation/parser` | 11 | Парсинг документов |
| `estimation/usecase` | 11 | CRUD, Submit, Aggregate |
| `shared/middleware` | 10 | JWT Auth, UserIDFromContext |
| `shared/pagination` | 13 | Parse, Offset |

## Версионирование

Проект следует [Semantic Versioning](https://semver.org/):

**Текущая версия: `0.5.0`**

### Changelog

#### v0.5.0 (2026-03-20)
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

MIT
