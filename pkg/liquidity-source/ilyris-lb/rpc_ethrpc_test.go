package ilyrislb

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ethrpc unpacks a single-tuple output into the target's first field; a bare lensPoolState
// target panicked on every live read (the fake-chain tests never reach this decode).
func TestLensPoolStateDecode(t *testing.T) {
	want := lensPoolState{
		Pool: common.HexToAddress("0x01"), TokenX: common.HexToAddress("0x02"), TokenY: common.HexToAddress("0x03"),
		DecimalsX: 18, DecimalsY: 6, BinStepBps: big.NewInt(10), SwapFeeBps: big.NewInt(1),
		ActiveId: big.NewInt(-7877), ActivePriceX18: big.NewInt(1), TotalFeeRate: big.NewInt(100000),
		BaseFeeRate: big.NewInt(1), ReserveX: big.NewInt(2), ReserveY: big.NewInt(3),
		ScannedFrom: big.NewInt(-1), ScannedTo: big.NewInt(1), PopulatedBins: big.NewInt(4),
		MarketGuard: common.HexToAddress("0x04"),
	}
	data, err := lensABI.Methods["getPoolState"].Outputs.Pack(want)
	if err != nil {
		t.Fatal(err)
	}
	var res struct{ State lensPoolState }
	if err := lensABI.UnpackIntoInterface(&res, "getPoolState", data); err != nil {
		t.Fatal(err)
	}
	if res.State.TokenY != want.TokenY || res.State.ActiveId.Int64() != -7877 || res.State.TotalFeeRate.Int64() != 100000 {
		t.Fatalf("decoded %+v", res.State)
	}
}
