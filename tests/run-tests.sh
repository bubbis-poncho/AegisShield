#!/bin/bash
# Test Runner Script for AegisShield
# Constitutional Principle: Comprehensive Testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="30m"
COVERAGE_THRESHOLD="90"
INTEGRATION_TIMEOUT="60m"

echo -e "${BLUE}🔬 AegisShield Test Suite${NC}"
echo "========================================"

# Function to run tests with proper formatting
run_test_suite() {
    local test_type="$1"
    local test_path="$2"
    local timeout="$3"
    
    echo -e "\n${YELLOW}Running $test_type tests...${NC}"
    
    if [ -d "$test_path" ]; then
        go test -v -timeout="$timeout" -race -cover "$test_path/..." || {
            echo -e "${RED}❌ $test_type tests failed${NC}"
            return 1
        }
        echo -e "${GREEN}✅ $test_type tests passed${NC}"
    else
        echo -e "${YELLOW}⚠️  $test_path not found - skipping $test_type tests${NC}"
    fi
}

# Function to check if services are running for integration tests
check_services() {
    echo -e "\n${YELLOW}Checking service availability...${NC}"
    
    services=("data-ingestion:9001" "entity-resolution:9002" "alerting-engine:9003" "graph-engine:9004")
    all_available=true
    
    for service in "${services[@]}"; do
        if nc -z ${service//:/ } 2>/dev/null; then
            echo -e "${GREEN}✅ $service is available${NC}"
        else
            echo -e "${RED}❌ $service is not available${NC}"
            all_available=false
        fi
    done
    
    if [ "$all_available" = false ]; then
        echo -e "${YELLOW}⚠️  Some services are not available. Integration tests may fail.${NC}"
        echo -e "${BLUE}ℹ️  Start services with: make dev-deploy${NC}"
        return 1
    fi
    
    return 0
}

# Main test execution
main() {
    local test_mode="${1:-all}"
    
    echo -e "${BLUE}Test mode: $test_mode${NC}"
    echo -e "${BLUE}Timeout: $TEST_TIMEOUT${NC}"
    echo -e "${BLUE}Coverage threshold: $COVERAGE_THRESHOLD%${NC}"
    
    case "$test_mode" in
        "unit")
            echo -e "\n${YELLOW}Running unit tests only...${NC}"
            run_test_suite "Unit" "./services/*/tests" "$TEST_TIMEOUT"
            ;;
        "integration")
            echo -e "\n${YELLOW}Running integration tests only...${NC}"
            check_services || exit 1
            run_test_suite "gRPC Contract" "./tests/integration/grpc-contracts" "$INTEGRATION_TIMEOUT"
            run_test_suite "API Contract" "./tests/integration/api-contracts" "$INTEGRATION_TIMEOUT"
            run_test_suite "Integration Workflow" "./tests/integration/workflows" "$INTEGRATION_TIMEOUT"
            ;;
        "tdd"|"contracts")
            echo -e "\n${YELLOW}Running TDD contract tests (should fail initially)...${NC}"
            echo -e "${BLUE}These tests are designed to fail until services are implemented${NC}"
            
            # Run with continue-on-error for TDD
            go test -v -timeout="$TEST_TIMEOUT" ./tests/integration/grpc-contracts/... || true
            go test -v -timeout="$TEST_TIMEOUT" ./tests/integration/api-contracts/... || true
            go test -v -timeout="$TEST_TIMEOUT" ./tests/integration/workflows/... || true
            
            echo -e "\n${GREEN}TDD contract tests completed${NC}"
            echo -e "${BLUE}Next step: Implement services to make these tests pass${NC}"
            ;;
        "e2e")
            echo -e "\n${YELLOW}Running end-to-end tests...${NC}"
            check_services || exit 1
            run_test_suite "End-to-End" "./tests/e2e" "$INTEGRATION_TIMEOUT"
            ;;
        "coverage")
            echo -e "\n${YELLOW}Running tests with coverage analysis...${NC}"
            go test -v -timeout="$TEST_TIMEOUT" -race -coverprofile=coverage.out ./...
            go tool cover -html=coverage.out -o coverage.html
            
            # Check coverage threshold
            coverage=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
            echo -e "\n${BLUE}Total coverage: ${coverage}%${NC}"
            
            if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
                echo -e "${GREEN}✅ Coverage meets threshold (${COVERAGE_THRESHOLD}%)${NC}"
            else
                echo -e "${RED}❌ Coverage below threshold (${COVERAGE_THRESHOLD}%)${NC}"
                exit 1
            fi
            ;;
        "all"|*)
            echo -e "\n${YELLOW}Running all tests...${NC}"
            
            # Unit tests first
            run_test_suite "Unit" "./services/*/tests" "$TEST_TIMEOUT"
            
            # Integration tests if services are available
            if check_services; then
                run_test_suite "gRPC Contract" "./tests/integration/grpc-contracts" "$INTEGRATION_TIMEOUT"
                run_test_suite "API Contract" "./tests/integration/api-contracts" "$INTEGRATION_TIMEOUT"
                run_test_suite "Integration Workflow" "./tests/integration/workflows" "$INTEGRATION_TIMEOUT"
                run_test_suite "End-to-End" "./tests/e2e" "$INTEGRATION_TIMEOUT"
            else
                echo -e "${YELLOW}⚠️  Skipping integration tests - services not available${NC}"
            fi
            ;;
    esac
}

# Help message
show_help() {
    echo "Usage: $0 [MODE]"
    echo ""
    echo "Modes:"
    echo "  all          Run all tests (default)"
    echo "  unit         Run unit tests only"
    echo "  integration  Run integration tests only"
    echo "  tdd          Run TDD contract tests (expected to fail initially)"
    echo "  e2e          Run end-to-end tests only"
    echo "  coverage     Run tests with coverage analysis"
    echo "  help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 tdd       # Run failing tests for TDD workflow"
    echo "  $0 unit      # Run unit tests only"
    echo "  $0 coverage  # Run with coverage analysis"
}

# Script entry point
if [ "$1" = "help" ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    show_help
    exit 0
fi

main "$@"