#!/bin/bash

# Скрипт для запуска одного конкретного интеграционного теста
# Usage: ./scripts/test-single.sh "TestName"
# Examples:
#   ./scripts/test-single.sh "TestAuthRateLimit"
#   ./scripts/test-single.sh "TestUserCRUD"

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

if [ $# -eq 0 ]; then
    echo -e "${RED}❌ Укажите имя теста${NC}"
    echo -e "${BLUE}Usage: $0 \"TestName\"${NC}"
    echo -e "${BLUE}Examples:${NC}"
    echo -e "${BLUE}  $0 \"TestAuthRateLimit\"${NC}"
    echo -e "${BLUE}  $0 \"TestUserCRUD\"${NC}"
    exit 1
fi

TEST_NAME="$1"

echo -e "${BLUE}🧪 Запускаем тест: ${YELLOW}$TEST_NAME${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"

# Запускаем тест
go test -tags integration -v main/tests/integration -run "$TEST_NAME" -timeout=10m

TEST_EXIT_CODE=$?

echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════${NC}"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ Тест $TEST_NAME прошел успешно!${NC}"
else
    echo -e "${RED}❌ Тест $TEST_NAME упал${NC}"
fi

exit $TEST_EXIT_CODE