# EstimatePro

Коллаборативная платформа для оценки проектов.

## Architecture

- **Monorepo**: `backend/` (Go 1.26) + `frontend/` (Next.js 16, TypeScript)
- **Backend**: Modular Monolith + Clean Architecture. Chi router, pgx + PostgreSQL 18, Redis 8, MinIO (S3)
- **Frontend**: Next.js 16 App Router, shadcn/ui, Tailwind CSS v4, TanStack Query, Zustand, next-intl (ru/en)
- **Full spec**: see PROJECT_FOUNDATION.md at project root or in Downloads

## Quick Start

```bash
# Infrastructure (OrbStack / Docker)
just dev-infra          # postgres:18, redis:8-alpine, minio

# Backend
just dev-backend        # Go server on :8080

# Frontend
just dev-frontend       # Next.js on :3000

# Run backend manually with ENV
cd backend && export $(grep -v '^#' ../.env | xargs) && go run ./cmd/server

# Migrations
just migrate-up

# Tests
just test-backend
```

## Current State (17 марта 2026)

### Infrastructure
- docker-compose.yml: postgres:18, redis:8-alpine, minio, **backend** (Go в контейнере)
- Все контейнеры работают в OrbStack. Frontend запускается нативно (`npm run dev`)
- Миграции: `migrations/001_init.sql` — users, workspaces, workspace_members, projects, project_members

### Backend (Go, :8080)
- Chi router, health check `/api/v1/health` ✅
- JWT middleware (stub), CORS, structured logging, graceful shutdown, config from ENV
- **Auth module**: domain + handler stubs (POST /login, /register, /refresh) — логика НЕ реализована
- **Project module**: domain + handler stubs (CRUD endpoints) — логика НЕ реализована
  - **Members usecase DONE** ✅: AddMember (role validation, permission check), RemoveMember (prevent last admin removal), ListMembers, ListMembersWithUsers. 19 table-driven tests
  - Members handler endpoints wired: GET/POST/DELETE /api/v1/projects/{id}/members
- Infra: pgx pool, Redis client, S3 (MinIO) client

### Frontend (Next.js 16, :3000)
- **Landing**: Three.js aurora shader (light/dark), навбар, hero, карточки фич, статистика, футер с конфетти easter egg
- **Auth UI**: login/register формы (OAuth + email/password) — бэкенд stubs
- **Dashboard**: Aceternity sidebar, project cards → clickable `<Link>` → project detail
- **Project Detail** (`/dashboard/projects/[id]`): 3 таба:
  - Overview: инфо-карточка + **Timeline** (Aceternity, scroll-animated, 5 шагов жизненного цикла)
  - Documents: заглушка "Скоро будет доступно"
  - Members: список участников с ролями, добавление/удаление (UI готов, бэкенд stub)
- **CreateProjectDialog**: shadcn Dialog + Button (radix-ui)
- **i18n**: next-intl, ru (default, без префикса), en (/en/). Смена языка без мерцания (useTransition)
- **Тема**: next-themes, ThemeToggle, aurora shader реагирует через MutationObserver
- **Установленные 21st.dev компоненты**: timeline.tsx, interactive-logs-table-shadcnui.tsx, filters.tsx, starfall-portfolio-landing.tsx, modem-animated-footer.tsx, Aceternity sidebar

### What's NOT done yet (see Taskmaster tasks):
- Auth usecase/repo (регистрация, логин, JWT, OAuth) — Task #1
- Auth frontend integration — Task #2
- Workspace module — Task #3
- Project CRUD usecase/repo — Task #4
- Project frontend integration — Task #5
- Document module (backend + frontend) — Tasks #6, #7
- Estimation module (backend + frontend) — Tasks #8, #9
- Notifications (email, telegram) — Task #10
- WebSocket real-time — Task #11
- Settings/Profile — Task #12
- OAuth (Google/GitHub) — Task #13
- CI/CD — Task #14

### Task Tracking
Задачи ведутся в `.taskmaster/tasks/tasks.json` (14 задач, 69 подзадач). Используй Taskmaster AI MCP для просмотра и обновления.

## Project Structure

```
backend/
  cmd/server/main.go              # Entry point, wiring all modules
  Dockerfile                      # Multi-stage Go build
  internal/
    config/config.go              # ENV config loading
    modules/
      auth/
        domain/                   # User entity, interfaces, errors
        handler/                  # HTTP stubs: POST /login, /register, /refresh + DTOs
      project/
        domain/
          project.go              # Project entity, Role methods (CanManageMembers, IsValid), MemberWithUser
          project_test.go         # 12 table-driven tests for Role methods
          interfaces.go           # ProjectRepository, MemberRepository (incl. ListByProjectWithUsers)
          events.go               # Domain events
          errors.go               # Domain errors
        usecase/
          member.go               # MemberUsecase: Add/Remove/List members (fully implemented)
          member_test.go          # 7 tests with mock repos
        handler/http.go           # CRUD stubs + real member endpoints (Add/Remove/List)
        repository/postgres.go    # PostgreSQL repo incl. ListByProjectWithUsers (JOIN)
    shared/
      middleware/                 # auth.go (JWT stub), cors.go, logger.go
      errors/                    # Standard HTTP error responses
      pagination/                # ?page=&limit= parsing
    infra/
      postgres/                  # pgx connection pool
      redis/                     # go-redis client
      s3/                        # MinIO client
  pkg/
    jwt/                         # JWT token generation/validation (stub)
  migrations/
    001_init.sql                 # Initial schema (goose): users, workspaces, projects, members

frontend/
  app/[locale]/
    page.tsx                     # Landing (PortfolioPage + Footer + confetti)
    layout.tsx                   # Root layout with NextIntlClientProvider
    (auth)/login/page.tsx        # Login page
    (auth)/register/page.tsx     # Register page
    dashboard/
      layout.tsx                 # Sidebar layout (Aceternity)
      page.tsx                   # Dashboard: project cards (Link), stats, CreateProjectDialog
      projects/[id]/page.tsx     # Project detail: Overview (Timeline) + Documents + Members tabs
      settings/page.tsx          # Settings
  components/ui/
    timeline.tsx                 # Aceternity timeline (scroll-animated, adapted for themes)
    interactive-logs-table-shadcnui.tsx  # Log table with filters, search, expand (for documents)
    filters.tsx                  # Linear-style filters (combobox, operators, chips)
    starfall-portfolio-landing.tsx  # Landing hero + aurora Three.js shader
    modem-animated-footer.tsx    # Footer с EP лого + onBrandClick
    sidebar.tsx                  # Aceternity sidebar
    locale-toggle.tsx            # Language switcher (ru/en, useTransition)
    theme-toggle.tsx             # Dark/light toggle
    button.tsx                   # shadcn Button (radix-ui, hover works)
    dialog.tsx, input.tsx, textarea.tsx, label.tsx, badge.tsx  # shadcn/ui
    popover.tsx, avatar.tsx, command.tsx, dropdown-menu.tsx, checkbox.tsx  # shadcn/ui (for filters)
  features/
    projects/
      api.ts                    # API functions: CRUD projects, members (listMembers, addMember, removeMember)
      components/
        create-project-dialog.tsx  # Dialog for creating projects
        add-member-dialog.tsx      # Dialog for adding members
        members-list.tsx           # Members list with roles, add/remove
  i18n/
    routing.ts                  # locales: ru (default), en; localePrefix: "as-needed"
    navigation.ts               # createNavigation(routing) — useRouter, usePathname, Link
    request.ts                  # next-intl server config
  lib/
    api-client.ts               # Fetch wrapper with auth headers
    query-client.ts             # TanStack Query provider
  messages/ru.json              # Russian translations (incl. timeline, roles, projects)
  messages/en.json              # English translations
```

## Key Rules

### Go Backend
- **Go 1.26** — use modern features: `new(val)`, `errors.AsType[T]`, `wg.Go()`, `t.Context()`, `omitzero`, `cmp.Or`, `slices`/`maps`/`cmp`
- **Dependency Rule**: infra → usecase → domain. Domain knows nothing about infra
- **No cross-module imports** — communicate via interfaces or domain events
- **No business logic in handlers**
- **Error wrapping**: `fmt.Errorf("scope.Method: %w", err)`
- **Table-driven tests**, `*_test.go` next to source
- **TDD mandatory**: estimation module (parser + aggregation), domain layers

### TypeScript / Next.js
- Strict TypeScript, no `any`
- All strings via `useTranslations()` — no hardcoded text in JSX
- `'use client'` only where interactivity is needed
- Server state: TanStack Query. UI state: Zustand. **Never** server data in Zustand
- Components: PascalCase, one per file
- API functions in `features/<module>/api/`

### Frontend Patterns
- **i18n navigation**: всегда использовать `useRouter`/`usePathname` из `@/i18n/navigation`, не из `next/navigation`
- **EP Logo**: везде `bg-primary` + `text-primary-foreground` (единый стиль)
- **Кнопки на лендинге**: текстовые, без бордеров — `text-muted-foreground hover:text-foreground transition-colors`
- **Aurora shader**: адаптивный light/dark через MutationObserver на `document.documentElement.classList`
- **21st.dev компоненты**: starfall-portfolio-landing, modem-animated-footer, Aceternity sidebar, Aceternity timeline, interactive-logs-table, Linear-style filters
- **PostgreSQL 18**: volume mount → `/var/lib/postgresql` (не `/var/lib/postgresql/data`)

### Git
- Branches: `feat/`, `fix/`, `chore/`
- Conventional commits: `feat: add file upload`, `fix: parser crash on empty pdf`

## ENV Variables

See `.env.example` in project root. Backend loads from ENV, run locally:
```bash
cd backend && export $(grep -v '^#' ../.env | xargs) && go run ./cmd/server
```
