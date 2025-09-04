#!/bin/bash

# Скрипт для поиска упавших интеграционных тестов
# Показывает только упавшие тесты с их ошибками

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Поиск упавших интеграционных тестов...${NC}"

# Временный файл для результатов
TEMP_FILE=$(mktemp)
FAILED_TESTS_FILE=$(mktemp)

# Запускаем тесты и сохраняем результат
echo -e "${YELLOW}Запускаем интеграционные тесты...${NC}"
go test -tags integration -v main/tests/integration 2>&1 | tee "$TEMP_FILE"

# Проверяем код возврата
TEST_EXIT_CODE=${PIPESTATUS[0]}

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "\n${GREEN}✅ Все тесты прошли успешно!${NC}"
    rm -f "$TEMP_FILE" "$FAILED_TESTS_FILE"
    exit 0
fi

echo -e "\n${RED}❌ Найдены упавшие тесты${NC}\n"

# Парсим упавшие тесты
grep -E "^--- FAIL:|FAIL\s+main/tests/integration|panic:" "$TEMP_FILE" | while read -r line; do
    echo "$line" >> "$FAILED_TESTS_FILE"
done

# Показываем упавшие тесты
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo -e "${RED}                        УПАВШИЕ ТЕСТЫ                        ${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}\n"

# Ищем тесты с конкретными ошибками
echo -e "${BLUE}🔍 Анализ ошибок:${NC}\n"

# 1. Privacy/Authentication ошибки
echo -e "${YELLOW}📋 Privacy/Authentication ошибки:${NC}"
grep -A 5 -B 2 "authentication required\|privacy.*deny\|access denied" "$TEMP_FILE" | \
    grep -E "^=== RUN|Error:|authentication required|privacy.*deny|access denied" | \
    head -20 || echo "  Не найдены"

echo ""

# 2. Field update ошибки
echo -e "${YELLOW}📋 Field update ошибки:${NC}"
grep -A 5 -B 2 "field.*not allowed\|cannot.*field" "$TEMP_FILE" | \
    grep -E "^=== RUN|Error:|field.*not allowed|cannot.*field" | \
    head -20 || echo "  Не найдены"

echo ""

# 3. Build ошибки
echo -e "${YELLOW}📋 Build/Compile ошибки:${NC}"
grep -A 3 -B 1 "build failed\|undefined:" "$TEMP_FILE" | \
    grep -E "build failed|undefined:|cannot find package" | \
    head -10 || echo "  Не найдены"

echo ""

# 4. Список всех упавших тестов
echo -e "${YELLOW}📋 Полный список упавших тестов:${NC}"
grep -E "^--- FAIL:" "$TEMP_FILE" | sed 's/^--- FAIL: /  ❌ /' || echo "  Не найдены через FAIL pattern"

# Альтернативный способ поиска упавших тестов
grep -E "FAIL\s+main/tests/integration" "$TEMP_FILE" | sed 's/^/  ❌ /' || echo ""

echo ""

# 5. Показываем статистику
echo -e "${YELLOW}📊 Статистика:${NC}"
TOTAL_TESTS=$(grep -c "^=== RUN" "$TEMP_FILE" 2>/dev/null || echo "0")
FAILED_TESTS=$(grep -c "^--- FAIL:" "$TEMP_FILE" 2>/dev/null || echo "0")
PASSED_TESTS=$((TOTAL_TESTS - FAILED_TESTS))

echo "  Всего тестов: $TOTAL_TESTS"
echo "  Прошли: $PASSED_TESTS"
echo "  Упали: $FAILED_TESTS"

echo ""
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}💡 Для подробной информации смотрите файл: $TEMP_FILE${NC}"
echo -e "${BLUE}💡 Чтобы запустить конкретный тест:${NC}"
echo -e "${BLUE}   make test-single TEST=\"TestName\"${NC}"
echo -e "${BLUE}💡 Или запустить тесты конкретного домена:${NC}"
echo -e "${BLUE}   make test-auth  # Тесты с префиксом auth_${NC}"
echo -e "${BLUE}   make test-user  # Тесты с префиксом user_${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"

# Очистка
rm -f "$FAILED_TESTS_FILE"

exit $TEST_EXIT_CODE