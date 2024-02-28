package ambient

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

// TODO: proxy call implementation
func callCrossFlag(poolHash string, tick types.Int24, isBuy bool, feeGlobal uint64) *big.Int {
	// function callCrossFlag (bytes32 poolHash, int24 tick,
	// 	bool isBuy, uint64 feeGlobal)
	// internal returns (int128 concLiqDelta) {
	// require(proxyPaths_[CrocSlots.FLAG_CROSS_PROXY_IDX] != address(0));

	// (bool success, bytes memory cmd) =
	// proxyPaths_[CrocSlots.FLAG_CROSS_PROXY_IDX].delegatecall
	// (abi.encodeWithSignature
	// ("crossCurveFlag(bytes32,int24,bool,uint64)",
	// poolHash, tick, isBuy, feeGlobal));
	// require(success);

	// concLiqDelta = abi.decode(cmd, (int128));
	// }

	return nil
}
