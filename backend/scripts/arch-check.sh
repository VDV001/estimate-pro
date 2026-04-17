#!/usr/bin/env bash
# arch-check.sh — архитектурные гейты для EstimatePro.
# Проверяет правила TDD+DDD+Clean Architecture, закреплённые в CLAUDE.md.
# Запускается локально (`just arch-check`) и в CI.
set -euo pipefail

cd "$(dirname "$0")/.."

violations=0

report_fail() {
    echo "  FAIL: $1"
    violations=$((violations + 1))
}

# Entities with domain constructors — прямое создание литералом запрещено
# вне domain/ и repository/ (repo reconstructs from DB rows). Расширять этот
# список по мере добавления NewXxx-конструкторов в других модулях.
entities_with_ctor='Workspace|Project|Member|Estimation|EstimationItem|User|Notification|Preference|DeliveryLog|Document|DocumentVersion|BotSession|MemoryEntry'

# --- Rule 1: domain — чистый, без инфры ---
echo "→ 1/5  domain/ не импортирует pgx/redis/chi/minio"
infra_in_domain=$(grep -rn --include='*.go' \
    -E 'github\.com/(jackc/pgx|redis/go-redis|go-chi|minio)' \
    internal/modules/*/domain/ 2>/dev/null || true)
if [ -n "$infra_in_domain" ]; then
    report_fail "инфра-либы в domain (должен быть pure):"
    echo "$infra_in_domain" | sed 's/^/    /'
fi

# --- Rule 2: нет cross-module импортов ---
echo "→ 2/5  modules/X не импортирует modules/Y"
for module_path in internal/modules/*/; do
    module=$(basename "$module_path")
    cross=$(grep -rn --include='*.go' \
        'github.com/VDV001/estimate-pro/backend/internal/modules/' \
        "$module_path" 2>/dev/null \
        | grep -v "internal/modules/$module/" || true)
    if [ -n "$cross" ]; then
        report_fail "cross-module импорт из $module:"
        echo "$cross" | sed 's/^/    /'
    fi
done

# --- Rule 3: handler не генерирует uuid/time (бизнес-логика) ---
echo "→ 3/5  handler не вызывает uuid.New()/time.Now() напрямую"
uuid_in_handler=$(grep -rn --include='*.go' --exclude='*_test.go' \
    -E 'uuid\.New\(\)|time\.Now\(\)' \
    internal/modules/*/handler/ 2>/dev/null || true)
if [ -n "$uuid_in_handler" ]; then
    filtered=$(echo "$uuid_in_handler" | grep -v '// arch:allow' || true)
    if [ -n "$filtered" ]; then
        report_fail "uuid.New()/time.Now() в handler (перенести в usecase, иначе пометить '// arch:allow TODO #X'):"
        echo "$filtered" | sed 's/^/    /'
    fi
fi

# --- Rule 4: handler не импортирует инфру напрямую ---
echo "→ 4/5  handler не импортирует pgx/redis/minio"
infra_in_handler=$(grep -rn --include='*.go' --exclude='*_test.go' \
    -E 'github\.com/(jackc/pgx|redis/go-redis|minio)' \
    internal/modules/*/handler/ 2>/dev/null || true)
if [ -n "$infra_in_handler" ]; then
    report_fail "инфра-импорт в handler (должен жить в repository):"
    echo "$infra_in_handler" | sed 's/^/    /'
fi

# --- Rule 5: domain-entity создаётся только через конструктор ---
echo "→ 5/5  &domain.(${entities_with_ctor}){} — только через NewXxx, вне domain/repository"
direct_literal=$(grep -rn --include='*.go' --exclude='*_test.go' \
    -E "&domain\\.(${entities_with_ctor})[[:space:]]*\\{" \
    internal/modules/ 2>/dev/null \
    | grep -v '/domain/' \
    | grep -v '/repository/' \
    | grep -v '/repo/' || true)
if [ -n "$direct_literal" ]; then
    report_fail "прямое создание domain-entity (использовать domain.NewXxx):"
    echo "$direct_literal" | sed 's/^/    /'
fi

echo
if [ "$violations" -gt 0 ]; then
    echo "arch-check: FAIL ($violations violations)"
    exit 1
fi
echo "arch-check: PASS"
