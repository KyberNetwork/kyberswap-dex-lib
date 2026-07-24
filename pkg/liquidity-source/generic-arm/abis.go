package genericarm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lidoArmABI abi.ABI
	// baseAssetConfigsV2ABI decodes the newer 9-field baseAssetConfigs(address) tuple (adds
	// baseAssetDecimals, widens pendingRedeemAssets to uint128) used by some AbstractARM deployments
	// (e.g. WETH_ARM). Older deployments (e.g. EthenaARM) return an 8-field tuple decoded by lidoArmABI
	// instead; see fetchAssetAndState's use of UnpackABI to try both.
	baseAssetConfigsV2ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lidoArmABI, lidoArmABIData},
		{&baseAssetConfigsV2ABI, baseAssetConfigsV2ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
