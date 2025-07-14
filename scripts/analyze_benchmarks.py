#!/usr/bin/env python3
"""
Benchmark Analysis Tool for cosmos/evm performance comparison

This script analyzes benchmark results and generates detailed performance reports
comparing the original cosmos/evm with the evmone-integrated version.
"""

import argparse
import os
import re
import statistics
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Tuple


@dataclass
class BenchmarkResult:
    """Represents a single benchmark result."""
    name: str
    iterations: int
    ns_per_op: float
    bytes_per_op: int
    allocs_per_op: int


@dataclass
class BenchmarkComparison:
    """Represents a comparison between two benchmark results."""
    name: str
    original: BenchmarkResult
    evmone: BenchmarkResult

    @property
    def speed_improvement(self) -> float:
        """Returns speed improvement as percentage (negative = slower)."""
        return ((self.original.ns_per_op - self.evmone.ns_per_op) / self.original.ns_per_op) * 100

    @property
    def memory_improvement(self) -> float:
        """Returns memory improvement as percentage (negative = worse)."""
        if self.original.bytes_per_op == 0:
            return 0.0
        return ((self.original.bytes_per_op - self.evmone.bytes_per_op) / self.original.bytes_per_op) * 100

    @property
    def alloc_improvement(self) -> float:
        """Returns allocation improvement as percentage (negative = worse)."""
        if self.original.allocs_per_op == 0:
            return 0.0
        return ((self.original.allocs_per_op - self.evmone.allocs_per_op) / self.original.allocs_per_op) * 100


class BenchmarkParser:
    """Parser for Go benchmark results."""

    BENCHMARK_PATTERN = re.compile(
        r'Benchmark(\w+)(?:-\d+)?\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?'
    )

    def parse_file(self, filepath: str) -> List[BenchmarkResult]:
        """Parse benchmark results from a file."""
        results = []

        with open(filepath, 'r') as f:
            content = f.read()

        for match in self.BENCHMARK_PATTERN.finditer(content):
            name = match.group(1)
            iterations = int(match.group(2))
            ns_per_op = float(match.group(3))
            bytes_per_op = int(match.group(4)) if match.group(4) else 0
            allocs_per_op = int(match.group(5)) if match.group(5) else 0

            results.append(BenchmarkResult(
                name=name,
                iterations=iterations,
                ns_per_op=ns_per_op,
                bytes_per_op=bytes_per_op,
                allocs_per_op=allocs_per_op
            ))

        return results

    def aggregate_results(self, results: List[BenchmarkResult]) -> Dict[str, BenchmarkResult]:
        """Aggregate multiple runs of the same benchmark."""
        grouped = {}

        for result in results:
            if result.name not in grouped:
                grouped[result.name] = []
            grouped[result.name].append(result)

        aggregated = {}
        for name, group in grouped.items():
            # Use median for more stable results
            ns_per_ops = [r.ns_per_op for r in group]
            bytes_per_ops = [r.bytes_per_op for r in group]
            allocs_per_ops = [r.allocs_per_op for r in group]

            aggregated[name] = BenchmarkResult(
                name=name,
                iterations=group[0].iterations,  # Assume same for all runs
                ns_per_op=statistics.median(ns_per_ops),
                bytes_per_op=int(statistics.median(bytes_per_ops)),
                allocs_per_op=int(statistics.median(allocs_per_ops))
            )

        return aggregated


class ReportGenerator:
    """Generates performance analysis reports."""

    def __init__(self):
        self.parser = BenchmarkParser()

    def generate_comparison_report(self, original_file: str, evmone_file: str, output_file: str) -> None:
        """Generate a detailed comparison report."""
        original_results = self.parser.aggregate_results(self.parser.parse_file(original_file))
        evmone_results = self.parser.aggregate_results(self.parser.parse_file(evmone_file))

        comparisons = []
        for name in original_results:
            if name in evmone_results:
                comparisons.append(BenchmarkComparison(
                    name=name,
                    original=original_results[name],
                    evmone=evmone_results[name]
                ))

        with open(output_file, 'w') as f:
            self._write_report(f, comparisons)

    def _write_report(self, f, comparisons: List[BenchmarkComparison]) -> None:
        """Write the comparison report to a file."""
        f.write("# EVM Performance Analysis Report\n\n")
        f.write(f"Generated on: {self._get_timestamp()}\n\n")

        # Executive Summary
        f.write("## Executive Summary\n\n")
        self._write_summary(f, comparisons)

        # Detailed Results
        f.write("\\n## Detailed Performance Analysis\n\n")
        self._write_detailed_results(f, comparisons)

        # Performance Categories
        f.write("\\n## Performance by Category\n\n")
        self._write_category_analysis(f, comparisons)

        # Recommendations
        f.write("\\n## Recommendations\n\n")
        self._write_recommendations(f, comparisons)

    def _write_summary(self, f, comparisons: List[BenchmarkComparison]) -> None:
        """Write executive summary."""
        if not comparisons:
            f.write("No comparable benchmarks found.\\n\\n")
            return

        speed_improvements = [c.speed_improvement for c in comparisons]
        memory_improvements = [c.memory_improvement for c in comparisons]
        alloc_improvements = [c.alloc_improvement for c in comparisons]

        avg_speed = statistics.mean(speed_improvements)
        avg_memory = statistics.mean(memory_improvements)
        avg_alloc = statistics.mean(alloc_improvements)

        f.write(f"**Performance Summary ({len(comparisons)} benchmarks analyzed):**\n\n")
        f.write(f"- Average Speed Change: {avg_speed:+.1f}% ({'Faster' if avg_speed > 0 else 'Slower'})\n")
        f.write(f"- Average Memory Change: {avg_memory:+.1f}% ({'Better' if avg_memory > 0 else 'Worse'})\n")
        f.write(f"- Average Allocation Change: {avg_alloc:+.1f}% ({'Better' if avg_alloc > 0 else 'Worse'})\n\n")

        # Best performers
        best_speed = max(comparisons, key=lambda c: c.speed_improvement)
        worst_speed = min(comparisons, key=lambda c: c.speed_improvement)

        f.write(f"**Best Speed Improvement:** {best_speed.name} ({best_speed.speed_improvement:+.1f}%)\\n")
        f.write(f"**Worst Speed Change:** {worst_speed.name} ({worst_speed.speed_improvement:+.1f}%)\\n\\n")

    def _write_detailed_results(self, f, comparisons: List[BenchmarkComparison]) -> None:
        """Write detailed benchmark results table."""
        f.write("| Benchmark | Original (ns/op) | Evmone (ns/op) | Speed Change | Memory Change | Alloc Change |\n")
        f.write("|-----------|------------------|----------------|--------------|---------------|---------------|\n")

        for comp in sorted(comparisons, key=lambda c: c.speed_improvement, reverse=True):
            speed_emoji = "🚀" if comp.speed_improvement > 10 else "⚡" if comp.speed_improvement > 0 else "🐌" if comp.speed_improvement < -10 else "→"

            f.write(f"| {comp.name} | {comp.original.ns_per_op:,.0f} | {comp.evmone.ns_per_op:,.0f} | ")
            f.write(f"{speed_emoji} {comp.speed_improvement:+.1f}% | {comp.memory_improvement:+.1f}% | {comp.alloc_improvement:+.1f}% |\n")

        f.write("\\n")

    def _write_category_analysis(self, f, comparisons: List[BenchmarkComparison]) -> None:
        """Write analysis by performance categories."""
        categories = {
            'Apply': [c for c in comparisons if 'Apply' in c.name],
            'Token': [c for c in comparisons if 'Token' in c.name],
            'Message': [c for c in comparisons if 'Message' in c.name],
            'Other': [c for c in comparisons if not any(x in c.name for x in ['Apply', 'Token', 'Message'])]
        }

        for category, benchmarks in categories.items():
            if not benchmarks:
                continue

            f.write(f"### {category} Operations\n\n")
            avg_speed = statistics.mean([b.speed_improvement for b in benchmarks])
            f.write(f"Average speed improvement: {avg_speed:+.1f}%\\n\\n")

            for bench in benchmarks:
                f.write(f"- **{bench.name}**: {bench.speed_improvement:+.1f}% speed, {bench.memory_improvement:+.1f}% memory\\n")
            f.write("\\n")

    def _write_recommendations(self, f, comparisons: List[BenchmarkComparison]) -> None:
        """Write performance recommendations."""
        significant_improvements = [c for c in comparisons if c.speed_improvement > 10]
        significant_regressions = [c for c in comparisons if c.speed_improvement < -10]

        if significant_improvements:
            f.write("### ✅ Significant Improvements\n\n")
            for comp in significant_improvements:
                f.write(f"- **{comp.name}**: {comp.speed_improvement:+.1f}% faster\\n")
            f.write("\\n")

        if significant_regressions:
            f.write("### ⚠️  Performance Regressions\n\n")
            for comp in significant_regressions:
                f.write(f"- **{comp.name}**: {comp.speed_improvement:+.1f}% slower (investigate)\\n")
            f.write("\\n")

        f.write("### 📊 Integration Decision\n\n")
        overall_improvement = statistics.mean([c.speed_improvement for c in comparisons])

        if overall_improvement > 5:
            f.write("**Recommendation: PROCEED** with evmone integration\\n")
            f.write(f"Overall performance improvement of {overall_improvement:.1f}% justifies integration.\\n")
        elif overall_improvement > -5:
            f.write("**Recommendation: EVALUATE** further\\n")
            f.write("Performance impact is neutral. Consider other factors like maintenance, features.\\n")
        else:
            f.write("**Recommendation: INVESTIGATE** performance issues\\n")
            f.write(f"Performance regression of {abs(overall_improvement):.1f}% needs investigation.\\n")

    def _get_timestamp(self) -> str:
        """Get current timestamp."""
        from datetime import datetime
        return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Analyze EVM benchmark results")
    parser.add_argument("--original", required=True, help="Original benchmark results file")
    parser.add_argument("--evmone", required=True, help="Evmone benchmark results file")
    parser.add_argument("--output", required=True, help="Output report file")
    parser.add_argument("--format", choices=["markdown", "text"], default="markdown", help="Output format")

    args = parser.parse_args()

    if not os.path.exists(args.original):
        print(f"Error: Original results file not found: {args.original}")
        sys.exit(1)

    if not os.path.exists(args.evmone):
        print(f"Error: Evmone results file not found: {args.evmone}")
        sys.exit(1)

    # Create output directory if needed
    os.makedirs(os.path.dirname(args.output), exist_ok=True)

    # Generate report
    generator = ReportGenerator()
    generator.generate_comparison_report(args.original, args.evmone, args.output)

    print(f"Analysis complete! Report saved to: {args.output}")


if __name__ == "__main__":
    main()