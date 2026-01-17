#!/bin/bash

# Скрипт для запуска тестов Task Manager
# Предоставляет удобный для чтения вывод при запуске тестов в приложении Task Manager

set -e  # Выход при любой ошибке

# Цветовые коды ANSI
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # Без цвета

# Печать цветного вывода
print_info() {
    echo -e "${BLUE}[ИНФО]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[УСПЕШНО]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[ПРЕДУПРЕЖДЕНИЕ]${NC} $1"
}

print_error() {
    echo -e "${RED}[ОШИБКА]${NC} $1"
}

print_section() {
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
}

# Функция для запуска тестов с удобочитаемым выводом
run_tests() {
    local test_type=$1
    local description=$2

    print_info "Запуск $description тестов..."

    case $test_type in
        "unit")
            go test ./internal/... -short -v 2>&1 | tee /tmp/unit_test_results.log
            ;;
        "integration")
            go test ./... -tags=integration -v 2>&1 | tee /tmp/integration_test_results.log
            ;;
        "usecase")
            go test ./internal/usecase/ -v 2>&1 | tee /tmp/usecase_test_results.log
            ;;
        "domain")
            go test ./internal/entity/domain/ -v 2>&1 | tee /tmp/domain_test_results.log
            ;;
        "handlers")
            go test ./internal/handlers/... -v 2>&1 | tee /tmp/handlers_test_results.log
            ;;
        "infrastructure")
            go test ./internal/infrastructure/... -v 2>&1 | tee /tmp/infrastructure_test_results.log
            ;;
        "all")
            go test ./... -v 2>&1 | tee /tmp/all_test_results.log
            ;;
        *)
            print_error "Неизвестный тип теста: $test_type"
            print_info "Доступные типы тестов: unit, integration, usecase, domain, handlers, infrastructure, all"
            exit 1
            ;;
    esac

    local exit_code=${PIPESTATUS[0]}

    if [ $exit_code -eq 0 ]; then
        print_success "$description тесты успешно завершены!"
    else
        print_error "$description тесты не прошли!"
        exit $exit_code
    fi
}

# Функция для отображения покрытия тестами
show_coverage() {
    local package=$1
    local output_file="coverage.html"

    if [ -z "$package" ]; then
        package="./..."
    fi

    print_info "Генерация отчета о покрытии тестами для $package..."

    go test -coverprofile=coverage.out $package
    if [ $? -eq 0 ]; then
        go tool cover -html=coverage.out -o $output_file
        print_success "Отчет о покрытии сгенерирован: $output_file"
        print_info "Открытие отчета о покрытии в браузере (если доступно)..."

        # Попытка открыть отчет о покрытии в браузере
        if command -v xdg-open &> /dev/null; then
            xdg-open $output_file
        elif command -v open &> /dev/null; then
            open $output_file
        else
            print_info "Пожалуйста, откройте $output_file в браузере для просмотра покрытия."
        fi
    else
        print_error "Не удалось сгенерировать отчет о покрытии"
    fi
}

# Функция для запуска тестов с покрытием
run_tests_with_coverage() {
    local package=$1

    if [ -z "$package" ]; then
        package="./..."
    fi

    print_info "Запуск тестов с покрытием для $package..."

    go test -coverprofile=coverage.out -v $package
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        print_success "Тесты успешно завершены с покрытием!"
        print_info "Резюме покрытия:"
        go tool cover -func=coverage.out
    else
        print_error "Тесты не прошли!"
    fi

    return $exit_code
}

# Функция для запуска конкретного теста по имени
run_specific_test() {
    local test_name=$1
    local package=$2

    if [ -z "$package" ]; then
        package="./..."
    fi

    print_info "Запуск конкретного теста '$test_name' в пакете '$package'..."

    go test -run "$test_name" -v $package
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        print_success "Тест '$test_name' успешно завершен!"
    else
        print_error "Тест '$test_name' не прошел!"
    fi

    return $exit_code
}

# Функция для перечисления всех доступных тестов
list_tests() {
    local package=$1

    if [ -z "$package" ]; then
        package="./..."
    fi

    print_info "Перечень доступных тестов в $package..."

    # Найти все файлы тестов и извлечь имена функций тестов
    find . -type f -name "*_test.go" -path "*/$package/*" -exec grep -h "^func Test" {} \; | sort -u | sed 's/^func \(Test[^ ]*\).*/\1/'
}

# Функция для отображения справки
show_help() {
    echo "Запускатель тестов Task Manager"
    echo ""
    echo "Использование: $0 [ОПЦИИ] [АРГУМЕНТЫ]"
    echo ""
    echo "Опции:"
    echo "  unit                     Запустить юнит-тесты"
    echo "  integration              Запустить интеграционные тесты"
    echo "  usecase                  Запустить тесты юзкейсов"
    echo "  domain                   Запустить тесты домена"
    echo "  handlers                 Запустить тесты обработчиков"
    echo "  infrastructure           Запустить тесты инфраструктуры"
    echo "  all                      Запустить все тесты"
    echo "  coverage [package]       Показать покрытие тестами для пакета (по умолчанию: все)"
    echo "  coverage-run [package]   Запустить тесты с покрытием для пакета (по умолчанию: все)"
    echo "  run [test-name] [package] Запустить конкретный тест по имени в пакете (по умолчанию: все)"
    echo "  list [package]          Перечислить все доступные тесты в пакете (по умолчанию: все)"
    echo "  help                     Показать это справочное сообщение"
    echo ""
    echo "Примеры:"
    echo "  $0 unit                          # Запустить юнит-тесты"
    echo "  $0 coverage ./internal/usecase/  # Показать покрытие для пакета usecase"
    echo "  $0 run TestCreateTask            # Запустить конкретный тест"
    echo "  $0 list ./internal/entity/       # Перечислить тесты в пакете entity"
}

# Основная логика скрипта
main() {
    if [ $# -eq 0 ]; then
        show_help
        exit 1
    fi

    case $1 in
        "unit")
            run_tests "unit" "Юнит"
            ;;
        "integration")
            run_tests "integration" "Интеграционные"
            ;;
        "usecase")
            run_tests "usecase" "Юзкейсы"
            ;;
        "domain")
            run_tests "domain" "Домен"
            ;;
        "handlers")
            run_tests "handlers" "Обработчики"
            ;;
        "infrastructure")
            run_tests "infrastructure" "Инфраструктура"
            ;;
        "all")
            run_tests "all" "Все"
            ;;
        "coverage")
            show_coverage "$2"
            ;;
        "coverage-run")
            run_tests_with_coverage "$2"
            ;;
        "run")
            if [ -z "$2" ]; then
                print_error "Пожалуйста, укажите имя теста для запуска"
                exit 1
            fi
            run_specific_test "$2" "$3"
            ;;
        "list")
            list_tests "$2"
            ;;
        "help")
            show_help
            ;;
        *)
            print_error "Неизвестная команда: $1"
            show_help
            exit 1
            ;;
    esac
}

# Запуск основной функции со всеми аргументами
main "$@"