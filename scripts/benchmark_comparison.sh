#!/bin/bash

# Benchmark Comparison Script for evmone vs original cosmos/evm
# This script compares performance between the original cosmos/evm and evmone-integrated version

set -e

# Configuration
BENCHMARK_RESULTS_DIR="benchmark_results"
ORIGINAL_BRANCH="test/original"
EVMONE_BRANCH="poc/evmone"
BENCHMARK_DURATION="2s"
BENCHMARK_COUNT=1

# Create results directory
mkdir -p $BENCHMARK_RESULTS_DIR

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_banner() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  EVM Performance Benchmark Comparison${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_benchmarks() {
    local branch_name=$1
    local results_file=$2

    print_step "Running benchmarks on branch: $branch_name"
    
    # Change to evmd directory if not already there
    if [[ $(basename "$PWD") != "evmd" ]]; then
        cd ./evmd
    fi
    # Core EVM benchmarks - targeting evmd integration tests
    print_info "Running core EVM benchmarks..."
    go test -run=^$ -bench=BenchmarkApply -benchtime=$BENCHMARK_DURATION -count=$BENCHMARK_COUNT \
        ./tests/integration/ \
        -benchmem \
        -tags=test \
        > "$results_file" 2>&1

    # Token operation benchmarks
    print_info "Running token operation benchmarks..."
    go test -run=^$ -bench=BenchmarkToken -benchtime=$BENCHMARK_DURATION -count=$BENCHMARK_COUNT \
        ./tests/integration/ \
        -benchmem \
        -tags=test \
        >> "$results_file" 2>&1

    # Message call benchmarks
    print_info "Running message call benchmarks..."
    go test -run=^$ -bench=BenchmarkMessage -benchtime=$BENCHMARK_DURATION -count=$BENCHMARK_COUNT \
        ./tests/integration/ \
        -benchmem \
        -tags=test \
        >> "$results_file" 2>&1

    # EmitLogs benchmarks
    print_info "Running emit logs benchmarks..."
    go test -run=^$ -bench=BenchmarkEmitLogs -benchtime=$BENCHMARK_DURATION -count=$BENCHMARK_COUNT \
        ./tests/integration/ \
        -benchmem \
        -tags=test \
        >> "$results_file" 2>&1
}

check_branch_exists() {
    if ! git rev-parse --verify "$1" >/dev/null 2>&1; then
        print_error "Branch '$1' does not exist"
        exit 1
    fi
}

backup_current_state() {
    print_step "Backing up current state"
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    CURRENT_COMMIT=$(git rev-parse HEAD)

    # Check for uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        print_error "You have uncommitted changes. Please commit or stash them before running benchmarks."
        exit 1
    fi
}

restore_state() {
    print_step "Restoring original state"
    git checkout "$CURRENT_BRANCH" >/dev/null 2>&1
}

benchmark_original() {
    print_step "Switching to original branch: $ORIGINAL_BRANCH"
    git checkout "$ORIGINAL_BRANCH" >/dev/null 2>&1

    print_info "Building original version..."
    make build >/dev/null 2>&1

    run_benchmarks "$ORIGINAL_BRANCH" "../$BENCHMARK_RESULTS_DIR/original_results.txt"
    print_success "Original benchmarks completed"
}

benchmark_evmone() {
    print_step "Switching to evmone branch: $EVMONE_BRANCH"
    git checkout "$EVMONE_BRANCH" >/dev/null 2>&1

    print_info "Building evmone version..."
    make build >/dev/null 2>&1

    run_benchmarks "$EVMONE_BRANCH" "../$BENCHMARK_RESULTS_DIR/evmone_results.txt"
    print_success "Evmone benchmarks completed"
}

analyze_results() {
    print_step "Analyzing benchmark results"

    if command -v benchcmp >/dev/null 2>&1; then
        print_info "Generating comparison report with benchcmp..."
        benchcmp "$BENCHMARK_RESULTS_DIR/original_results.txt" "$BENCHMARK_RESULTS_DIR/evmone_results.txt" > "$BENCHMARK_RESULTS_DIR/comparison_report.txt"

        echo -e "${GREEN}Comparison Report:${NC}"
        cat "$BENCHMARK_RESULTS_DIR/comparison_report.txt"
    else
        print_info "Installing benchcmp for detailed analysis..."
        go install golang.org/x/tools/cmd/benchcmp@latest
        if command -v benchcmp >/dev/null 2>&1; then
            benchcmp "$BENCHMARK_RESULTS_DIR/original_results.txt" "$BENCHMARK_RESULTS_DIR/evmone_results.txt" > "$BENCHMARK_RESULTS_DIR/comparison_report.txt"
            echo -e "${GREEN}Comparison Report:${NC}"
            cat "$BENCHMARK_RESULTS_DIR/comparison_report.txt"
        else
            print_error "Could not install benchcmp. Please install manually: go install golang.org/x/tools/cmd/benchcmp@latest"
        fi
    fi

    # Generate summary
    cat > "$BENCHMARK_RESULTS_DIR/summary.md" << EOF
# EVM Performance Benchmark Results

## Test Configuration
- Benchmark Duration: $BENCHMARK_DURATION
- Benchmark Count: $BENCHMARK_COUNT per test
- Original Branch: $ORIGINAL_BRANCH
- Evmone Branch: $EVMONE_BRANCH
- Test Date: $(date)

## Raw Results

### Original Implementation
\`\`\`
$(cat "$BENCHMARK_RESULTS_DIR/original_results.txt")
\`\`\`

### Evmone Implementation
\`\`\`
$(cat "$BENCHMARK_RESULTS_DIR/evmone_results.txt")
\`\`\`

$(if [ -f "$BENCHMARK_RESULTS_DIR/comparison_report.txt" ]; then
    echo "## Performance Comparison"
    echo "\`\`\`"
    cat "$BENCHMARK_RESULTS_DIR/comparison_report.txt"
    echo "\`\`\`"
fi)

## Key Metrics Analyzed
- **ApplyTransaction**: Core transaction execution performance
- **ApplyMessage**: Message processing efficiency
- **TokenTransfer**: ERC20 token transfer operations
- **TokenMint**: Token minting performance
- **MessageCall**: Contract-to-contract call performance
- **EmitLogs**: Event emission efficiency

## Interpretation Guide
- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)
- Performance improvements are shown as negative percentages
- Memory usage improvements are shown as negative percentages
EOF

    print_success "Results saved to $BENCHMARK_RESULTS_DIR/"
}

show_summary() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Benchmark Comparison Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${BLUE}Results Location:${NC} $BENCHMARK_RESULTS_DIR/"
    echo -e "${BLUE}Summary Report:${NC} $BENCHMARK_RESULTS_DIR/summary.md"
    echo -e "${BLUE}Raw Results:${NC}"
    echo -e "  - Original: $BENCHMARK_RESULTS_DIR/original_results.txt"
    echo -e "  - Evmone: $BENCHMARK_RESULTS_DIR/evmone_results.txt"
    if [ -f "$BENCHMARK_RESULTS_DIR/comparison_report.txt" ]; then
        echo -e "${BLUE}Comparison:${NC} $BENCHMARK_RESULTS_DIR/comparison_report.txt"
    fi
    echo ""
    echo -e "${YELLOW}Quick View:${NC}"
    if [ -f "$BENCHMARK_RESULTS_DIR/comparison_report.txt" ]; then
        head -20 "$BENCHMARK_RESULTS_DIR/comparison_report.txt"
    else
        echo "Run 'go install golang.org/x/tools/cmd/benchcmp@latest' for detailed comparison"
    fi
}

cleanup_on_error() {
    print_error "An error occurred. Cleaning up..."
    restore_state
    exit 1
}

# Main execution
main() {
    # Set up error handling
    trap cleanup_on_error ERR

    print_banner

    # Validate environment
    check_branch_exists "$ORIGINAL_BRANCH"
    check_branch_exists "$EVMONE_BRANCH"

    # Backup current state
    backup_current_state

    # Run benchmarks
    benchmark_original
    benchmark_evmone

    # Restore original state
    restore_state

    # Analyze results
    analyze_results

    # Show summary
    show_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            BENCHMARK_DURATION="$2"
            shift 2
            ;;
        --count)
            BENCHMARK_COUNT="$2"
            shift 2
            ;;
        --original-branch)
            ORIGINAL_BRANCH="$2"
            shift 2
            ;;
        --evmone-branch)
            EVMONE_BRANCH="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --duration DURATION    Benchmark duration (default: 30s)"
            echo "  --count COUNT          Number of benchmark runs (default: 5)"
            echo "  --original-branch BRANCH  Original branch name (default: main)"
            echo "  --evmone-branch BRANCH     Evmone branch name (default: poc/evm1)"
            echo "  --help                 Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main "$@"