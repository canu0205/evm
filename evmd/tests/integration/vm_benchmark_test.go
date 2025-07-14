package integration

import (
	"testing"

	"github.com/cosmos/evm/tests/integration/x/vm"
)

// State transition benchmarks
func BenchmarkApplyTransaction(b *testing.B) {
	vm.BenchmarkApplyTransaction(b, CreateEvmd)
}

func BenchmarkApplyMessage(b *testing.B) {
	vm.BenchmarkApplyMessage(b, CreateEvmd)
}

//func BenchmarkSetParams(b *testing.B) {
//}
//
//func BenchmarkGetParams(b *testing.B) {
//}
//
//// Token operation benchmarks - using the simplified pattern
//func BenchmarkTokenTransfer(b *testing.B) {
//}
//
//func BenchmarkTokenMint(b *testing.B) {
//}
//
//func BenchmarkMessageCall(b *testing.B) {
//}
//
//func BenchmarkEmitLogs(b *testing.B) {
//}
