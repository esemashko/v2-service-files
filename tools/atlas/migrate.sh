#!/bin/bash
set -e

# Функция логирования
log() {
    local message="[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    echo "$message"
    if [ -n "$LOG_FILE" ]; then
        echo "$message" >> "$LOG_FILE"
    fi
}

# Отладочный вывод
debug_log() {
    if [ "$DEBUG" = true ]; then
        log "DEBUG: $1"
    fi
}

# Корень проекта
if [ -z "$PROJECT_ROOT" ]; then
    if [ -d "/app" ]; then
        PROJECT_ROOT="/app"
    else
        PROJECT_ROOT="$(dirname "$0")/../.."
    fi
fi
cd "${PROJECT_ROOT}"

# Директория бэкапов
if [ -d "/backups" ]; then
    BACKUP_DIR="/backups"
else
    BACKUP_DIR="${PROJECT_ROOT}/backups"
fi

# Флаги
SKIP_BACKUP=false
DRY_RUN=false
CHECK_ONLY=false
DEBUG=false
FORCE_RECREATE=false
ROLLBACK_MODE=false
LOG_FILE=""

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-backup)
            SKIP_BACKUP=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --check)
            CHECK_ONLY=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --force-recreate)
            FORCE_RECREATE=true
            shift
            ;;
        --rollback)
            ROLLBACK_MODE=true
            shift
            ;;
        --log-file)
            LOG_FILE="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [--check|--rollback|--dry-run] [--skip-backup] [--force-recreate] [--debug] [--log-file <file>]"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"; exit 1
            ;;
    esac
done

# Загрузка .env
if [ -f .env ]; then
    # Загружаем переменные, правильно обрабатывая кавычки
    set -a  # Автоматически экспортировать все переменные
    source .env
    set +a  # Отключить автоэкспорт
    debug_log ".env loaded"
fi

# Значения по умолчанию
DB_SCHEMA=${DB_SCHEMA:-public}
if [ -z "$DB_SSLMODE" ]; then
    DB_SSLMODE=disable
fi
debug_log "DB_SCHEMA: ${DB_SCHEMA}"
debug_log "DB_SSLMODE: ${DB_SSLMODE}"

# Выбор хоста/порта: mutation -> query -> localhost
DB_HOST=${DB_MUTATION_HOST:-${DB_QUERY_HOST:-localhost}}
DB_PORT=${DB_MUTATION_PORT:-${DB_QUERY_PORT:-5432}}

# Проверка обязательных переменных
if [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
    echo "Error: DB_USER, DB_PASSWORD and DB_NAME must be set"; exit 1
fi

# URL-энкодинг
urlencode() {
    local LANG=C
    local length="${#1}"
    for (( i = 0; i < length; i++ )); do
        local c="${1:i:1}"
        case $c in
            [a-zA-Z0-9.~_-]) printf "$c" ;;
            *) printf '%%%02X' "'${c}" ;;
        esac
    done
}

ENCODED_PASSWORD=$(urlencode "$DB_PASSWORD")
DB_URL="postgres://${DB_USER}:${ENCODED_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
debug_log "DB_URL: ${DB_URL}"

# Бэкап схемы
backup_schema() {
    local schema=$1
    local backup_dir="${BACKUP_DIR}/$(date +'%Y%m%d')"
    local backup_file="${backup_dir}/${schema}_$(date +'%H%M%S').sql"
    mkdir -p "$backup_dir" || { echo "Warning: Cannot create backup directory ${backup_dir}, skipping backup"; return 0; }
    PGPASSWORD="${DB_PASSWORD}" pg_dump \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -n "$schema" \
        -f "$backup_file" || return 1
    log "Created backup: $backup_file"
}

# Пересоздание схемы (опасная операция, требует --force-recreate)
recreate_schema() {
    local schema=$1
    log "Force recreating schema: $schema"
    PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" -p "${DB_PORT}" \
        -U "${DB_USER}" -d "${DB_NAME}" \
        -c "DROP SCHEMA IF EXISTS ${schema} CASCADE; CREATE SCHEMA ${schema};" || return 1
}

# Проверка состояния
check_schema_status() {
    atlas migrate status \
        --dir "file://ent/migrate/migrations" \
        --url "${DB_URL}?search_path=${DB_SCHEMA}&sslmode=${DB_SSLMODE}"
}

# Применение миграций
apply_migrations() {
    if [ "$DRY_RUN" = true ]; then
        log "DRY-RUN: showing status for schema ${DB_SCHEMA}"
        check_schema_status
        return 0
    fi

    if [ "$FORCE_RECREATE" = true ]; then
        if [ "$SKIP_BACKUP" = false ]; then
            log "Creating backup before recreation for schema: ${DB_SCHEMA}"
            backup_schema "$DB_SCHEMA" || { log "Backup failed"; return 1; }
        fi
        recreate_schema "$DB_SCHEMA" || { log "Recreate failed"; return 1; }
    elif [ "$SKIP_BACKUP" = false ]; then
        log "Creating backup for schema: ${DB_SCHEMA}"
        backup_schema "$DB_SCHEMA" || { log "Backup failed"; return 1; }
    fi

    atlas migrate apply \
        --dir "file://ent/migrate/migrations" \
        --url "${DB_URL}?search_path=${DB_SCHEMA}&sslmode=${DB_SSLMODE}" \
        --allow-dirty
}

# Откат миграций
rollback_migrations() {
    if [ "$DRY_RUN" = true ]; then
        log "DRY-RUN: would rollback schema ${DB_SCHEMA}"
        return 0
    fi
    if [ "$SKIP_BACKUP" = false ]; then
        log "Creating backup before rollback for schema: ${DB_SCHEMA}"
        backup_schema "$DB_SCHEMA" || { log "Backup failed"; return 1; }
    fi
    atlas migrate rollback \
        --dir "file://ent/migrate/migrations" \
        --url "${DB_URL}?search_path=${DB_SCHEMA}&sslmode=${DB_SSLMODE}"
}

# Режимы
if [ "$CHECK_ONLY" = true ]; then
    if check_schema_status; then
        log "Schema ${DB_SCHEMA} is up to date"
        exit 0
    else
        log "Schema ${DB_SCHEMA} needs migration"
        exit 1
    fi
fi

if [ "$ROLLBACK_MODE" = true ]; then
    if rollback_migrations; then
        log "Rollback completed"
        exit 0
    else
        log "Rollback failed"
    exit 1
fi
else
    if apply_migrations; then
        log "Migrations applied successfully"
        exit 0
    else
        log "Migration failed"
        exit 1
    fi
fi

 
