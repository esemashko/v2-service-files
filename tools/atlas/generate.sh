#!/bin/bash

# Переходим в корневую директорию проекта
cd "$(dirname "$0")/../.."

# Функция отладочного вывода
debug_log() {
    if [ "$DEBUG" = true ]; then
        echo "[DEBUG] $1"
    fi
}

# Устанавливаем значения по умолчанию
MANUAL_MIGRATION=false
MIGRATION_NAME=""

# Выводим справку по использованию
function show_help {
    echo "Usage: $0 [OPTIONS] <migration-name>"
    echo
    echo "Options:"
    echo "  --manual    Create an empty migration file for manual editing"
    echo
    echo "Examples:"
    echo "  $0 add_users_table               # Generate automatic migration based on schema changes"
    echo "  $0 --manual update_existing_data  # Create an empty migration file for manual editing"
    exit 1
}

# Разбор аргументов командной строки
while [[ $# -gt 0 ]]; do
    case "$1" in
        --manual)
            MANUAL_MIGRATION=true
            shift
            ;;
        --help)
            show_help
            ;;
        *)
            if [[ -z "$MIGRATION_NAME" ]]; then
                MIGRATION_NAME="$1"
                shift
            else
                echo "Error: Unknown argument '$1'"
                show_help
            fi
            ;;
    esac
done

# Проверяем, передано ли имя миграции
if [ -z "$MIGRATION_NAME" ]; then
    echo "Error: Migration name is required"
    show_help
fi

# Загружаем переменные окружения из .env файла
if [ -f .env ]; then
    # Загружаем переменные, правильно обрабатывая кавычки
    set -a  # Автоматически экспортировать все переменные
    source .env
    set +a  # Отключить автоэкспорт
    debug_log ".env loaded"
else
    echo "Error: .env file not found"
    exit 1
fi

# Устанавливаем значение по умолчанию для схемы, если оно не задано (по умолчанию public)
DB_SCHEMA=${DB_SCHEMA:-public}

# Устанавливаем sslmode по умолчанию согласно README (disable по умолчанию)
if [ -z "$DB_SSLMODE" ]; then
    DB_SSLMODE=disable
else
    DB_SSLMODE="$DB_SSLMODE"
fi

debug_log "Database sslmode: ${DB_SSLMODE}"
debug_log "Target schema: ${DB_SCHEMA}"

# Функция для url-энкодинга (bash)
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

# Определяем параметры подключения (предпочитаем write-хост, затем read)
DB_HOST=${DB_MUTATION_HOST:-${DB_QUERY_HOST:-localhost}}
DB_PORT=${DB_MUTATION_PORT:-${DB_QUERY_PORT:-5432}}

# Проверяем обязательные переменные окружения
if [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
    echo "Error: DB_USER, DB_PASSWORD and DB_NAME must be set in .env or environment"
    exit 1
fi

# Формируем DSN для подключения к эталонной базе данных
ENCODED_PASSWORD=$(urlencode "$DB_PASSWORD")
DB_URL="postgres://${DB_USER}:${ENCODED_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
debug_log "Using database URL: ${DB_URL}"
debug_log "Using reference schema: ${DB_SCHEMA}"

# Создаем директорию для миграций, если её нет
mkdir -p ent/migrate/migrations

# Текущая дата и время в формате yyyymmddhhmmss
TIMESTAMP=$(date "+%Y%m%d%H%M%S")
MIGRATION_FILENAME="${TIMESTAMP}_${MIGRATION_NAME}.sql"
MIGRATION_PATH="ent/migrate/migrations/${MIGRATION_FILENAME}"

debug_log "Migration file path: $MIGRATION_PATH"

if [ "$MANUAL_MIGRATION" = true ]; then
    # Создаем пустой файл миграции для ручного редактирования
    cat > "$MIGRATION_PATH" << EOF
-- Manual migration: ${MIGRATION_NAME}
-- Created at: $(date "+%Y-%m-%d %H:%M:%S")
--
-- Write your SQL statements below:

EOF
    echo "Created empty migration file: ${MIGRATION_PATH}"
    echo "Edit this file to add your custom SQL statements."
    
    # Добавляем файл в atlas с правильной контрольной суммой
    atlas migrate hash --dir "file://ent/migrate/migrations"
    debug_log "Manual migration: $MANUAL_MIGRATION"
else
    # Генерируем новую миграцию для выбранной схемы
    # Используем public схему для dev-url (чистая схема для сравнения)
    atlas migrate diff $MIGRATION_NAME \
        --dir "file://ent/migrate/migrations" \
        --to "ent://ent/schema" \
        --dev-url "${DB_URL}?search_path=public&sslmode=${DB_SSLMODE}"

    echo "Migration generated in ent/migrate/migrations/"
fi