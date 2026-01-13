package stabull

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestPoolDiscovery discovers all Stabull pools from factory contracts on each chain
// This test queries the factory's getCurve method with known token pairs to find deployed pools
func TestPoolDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping pool discovery test in short mode")
	}

	// Define chains and their RPC endpoints
	chains := []struct {
		name           string
		chainID        uint
		rpcURL         string
		factoryAddress string
	}{
		{
			name:           "Polygon",
			chainID:        137,
			rpcURL:         "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: FactoryAddresses["polygon"],
		},
		{
			name:           "Base",
			chainID:        8453,
			rpcURL:         "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: FactoryAddresses["base"],
		},
		{
			name:           "Ethereum",
			chainID:        1,
			rpcURL:         "https://eth-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: FactoryAddresses["ethereum"],
		},
	}

	// Common stablecoins and tokens that might be paired with USDC
	// USDC addresses per chain
	usdcAddresses := map[string]string{
		"Polygon":  "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC on Polygon PoS
		"Base":     "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
		"Ethereum": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC on Ethereum
	}

	// Known base tokens that have Stabull pools
	// These addresses are from official Stabull documentation: https://docs.stabull.finance/amm/contracts
	knownTokens := map[string][]struct {
		symbol  string
		address string
	}{
		"Polygon": {
			{symbol: "AUDF", address: "0xd2a530170D71a9Cfe1651Fb468E2B98F7Ed7456b"},
			{symbol: "BRZ", address: "0x4ed141110f6eeeaba9a1df36d8c26f684d2475dc"},
			{symbol: "COPM", address: "0x12050c705152931cFEe3DD56c52Fb09Dea816C23"},
			{symbol: "DAI", address: "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063"},
			{symbol: "EURS", address: "0xE111178A87A3BFf0c8d18DECBa5798827539Ae99"},
			{symbol: "NZDS", address: "0xFbBE4b730e1e77d02dC40fEdF9438E2802eab3B5"},
			{symbol: "OFD", address: "0x9cFb3B1b217b41C4E748774368099Dd8Dd7E89A1"},
			{symbol: "PAXG", address: "0x553d3D295e0f695B9228246232eDF400ed3560B5"},
			{symbol: "PHPC", address: "0x87a25dc121Db52369F4a9971F664Ae5e372CF69A"},
			{symbol: "TRYB", address: "0x4Fb71290Ac171E1d144F7221D882BECAc7196EB5"},
			{symbol: "USDT", address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"},
			{symbol: "XSGD", address: "0xDC3326e71D45186F113a2F448984CA0e8D201995"},
			{symbol: "ZCHF", address: "0x02567e4b14b25549331fCEe2B56c647A8bAB16FD"},
		},
		"Base": {
			{symbol: "AUDD", address: "0x449b3317a6d1efb1bc3ba0700c9eaa4ffff4ae65"},
			{symbol: "BRZ", address: "0xE9185Ee218cae427aF7B9764A011bb89FeA761B4"},
			{symbol: "EURC", address: "0x60a3e35cc302bfa44cb288bc5a4f316fdb1adb42"},
			{symbol: "MXNe", address: "0x269cae7dc59803e5c596c95756faeebb6030e0af"},
			{symbol: "TRYB", address: "0xFb8718a69aed7726AFb3f04D2Bd4bfDE1BdCb294"},
			{symbol: "ZARP", address: "0xb755506531786C8aC63B756BaB1ac387bACB0C04"},
			{symbol: "ZCHF", address: "0xd4dd9e2f021bb459d5a5f6c24c12fe09c5d45553"},
			{symbol: "OFD", address: "0x7479791022eb1030bbc3b09f6575c5db4ddc0b90"},
		},
		"Ethereum": {
			{symbol: "AUDD", address: "0x4cCe605eD955295432958d8951D0B176C10720d5"},
			{symbol: "EURS", address: "0xdB25f211AB05b1c97D595516F45794528a807ad8"},
			{symbol: "GYEN", address: "0xc08512927d12348f6620a698105e1baac6ecd911"},
			{symbol: "NZDS", address: "0xda446fad08277b4d2591536f204e018f32b6831c"},
			{symbol: "TRYB", address: "0x2c537e5624e4af88a7ae4060c022609376c8d0eb"},
		},
	}

	for _, chain := range chains {
		t.Run(chain.name, func(t *testing.T) {
			ctx := context.Background()
			client := ethrpc.New(chain.rpcURL)
			require.NotNil(t, client)

			// Set multicall contract
			client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

			t.Logf("\n=== %s (Chain ID: %d) ===", chain.name, chain.chainID)
			t.Logf("Factory: %s", chain.factoryAddress)

			usdcAddress := usdcAddresses[chain.name]
			tokens := knownTokens[chain.name]

			poolCount := 0
			for _, token := range tokens {
				// Query factory for pool address using getCurve(base, quote)
				poolAddress, err := getCurveAddress(ctx, client, chain.factoryAddress, token.address, usdcAddress)
				if err != nil {
					t.Logf("  %s/USDC: Error querying factory - %v", token.symbol, err)
					continue
				}

				if poolAddress == (common.Address{}) {
					t.Logf("  %s/USDC: No pool found", token.symbol)
					continue
				}

				poolCount++
				t.Logf("  %s/USDC: %s", token.symbol, poolAddress.Hex())

				// Optionally verify pool is functional by fetching reserves
				reserves, err := getPoolReserves(ctx, client, poolAddress.Hex())
				if err != nil {
					t.Logf("    ⚠️  Warning: Could not fetch reserves - %v", err)
				} else {
					t.Logf("    ✓ Reserves: %s / %s", reserves[0], reserves[1])
				}
			}

			t.Logf("\nTotal pools found on %s: %d\n", chain.name, poolCount)
		})
	}
}

// getCurveAddress queries the factory contract for a pool address
func getCurveAddress(ctx context.Context, client *ethrpc.Client, factoryAddress, baseToken, quoteToken string) (common.Address, error) {
	var poolAddress common.Address

	rpcRequest := client.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullFactoryABI,
		Target: factoryAddress,
		Method: factoryMethodGetCurve,
		Params: []interface{}{
			common.HexToAddress(baseToken),
			common.HexToAddress(quoteToken),
		},
	}, []interface{}{&poolAddress})

	_, err := rpcRequest.Call()
	if err != nil {
		return common.Address{}, err
	}

	return poolAddress, nil
}

// getPoolReserves fetches reserves from a pool to verify it's functional
func getPoolReserves(ctx context.Context, client *ethrpc.Client, poolAddress string) ([]*big.Int, error) {
	type LiquidityResult struct {
		Total      *big.Int
		Individual []*big.Int
	}

	var liquidityResult LiquidityResult

	rpcRequest := client.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	if len(liquidityResult.Individual) != 2 {
		return nil, fmt.Errorf("expected 2 reserves, got %d", len(liquidityResult.Individual))
	}

	return liquidityResult.Individual, nil
}
