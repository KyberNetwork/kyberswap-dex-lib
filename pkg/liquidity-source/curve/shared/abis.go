package shared

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ERC20ABI  abi.ABI
	cERC20ABI abi.ABI
	OracleABI abi.ABI

	// various ABIs for probing
	gammaABI      abi.ABI
	gammaABIBytes = []byte(`[{"stateMutability":"view","type":"function","name":"gamma","inputs":[],"outputs":[{"name":"","type":"uint256"}]}]`)
	// there are 2 variants of underlying_coins, with int128 and uint256 input respectively
	underlyingCoins128ABI   abi.ABI
	underlyingCoins128Bytes = []byte(`[{"name":"underlying_coins","outputs":[{"type":"address","name":"out"}],"inputs":[{"type":"int128","name":"arg0"}],"constant":true,"payable":false,"type":"function"}]`)
	underlyingCoins256ABI   abi.ABI
	underlyingCoins256Bytes = []byte(`[{"name":"underlying_coins","outputs":[{"type":"address","name":"out"}],"inputs":[{"type":"uint256","name":"arg0"}],"constant":true,"payable":false,"type":"function"}]`)

	addressProviderABI      abi.ABI
	addressProviderABIBytes = []byte(`[{"name":"get_address","outputs":[{"type":"address","name":""}],"inputs":[{"type":"uint256","name":"_id"}],"stateMutability":"view","type":"function"}]`)

	MainRegistryABI      abi.ABI
	mainRegistryABIBytes = []byte(`[{"stateMutability":"view","type":"function","name":"get_rates","inputs":[{"name":"_pool","type":"address"}],"outputs":[{"name":"","type":"uint256[8]"}]}]`)
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ERC20ABI, erc20ABIBytes},
		{&cERC20ABI, cerc20ABIBytes},
		{&OracleABI, oracleABIBytes},
		{&gammaABI, gammaABIBytes},
		{&underlyingCoins128ABI, underlyingCoins128Bytes},
		{&underlyingCoins256ABI, underlyingCoins256Bytes},
		{&addressProviderABI, addressProviderABIBytes},
		{&MainRegistryABI, mainRegistryABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
