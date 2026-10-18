package rseth

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestGetNewPoolStatePinsAssetReadsToPoolBlock(t *testing.T) {
	for _, withOverrides := range []bool{false, true} {
		name := "without-overrides"
		if withOverrides {
			name = "with-overrides"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ethAsset := ethcommon.HexToAddress(common.ETH)
				otherAsset := ethcommon.HexToAddress("0x0000000000000000000000000000000000000009")
				resolver := func(_ ethcommon.Address, method abi.Method, _ []any) ([]any, error) {
					values := make([]any, len(method.Outputs))
					for i, output := range method.Outputs {
						values[i] = ethrpcmock.Value(output.Type)
					}
					if method.Name == common.LRTConfigMethodGetSupportedAssetList {
						values[0] = []ethcommon.Address{ethAsset, otherAsset}
					}
					return values, nil
				}
				srv, recorder := ethrpcmock.NewWithResolver(t, resolver,
					common.LRTConfigABI, common.LRTDepositPoolABI, common.LRTOracleABI, common.Erc20ABI)
				client := ethrpc.New(srv.URL)
				tracker := NewPoolTracker(client)
				p := entity.Pool{Address: common.LRTDepositPool}
				var updated entity.Pool
				var err error
				if withOverrides {
					overrides := map[ethcommon.Address]gethclient.OverrideAccount{
						ethcommon.HexToAddress("0x000000000000000000000000000000000000000a"): {Balance: big.NewInt(99)},
					}
					updated, err = tracker.GetNewPoolStateWithOverrides(context.Background(), p,
						pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
				} else {
					updated, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
				}
				require.NoError(t, err)
				require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
				require.Len(t, updated.Tokens, 3)
				require.Equal(t, common.WETH, updated.Tokens[1].Address)
				ethrpcmock.RequireRequestsPinned(t, recorder, 2)
				ethrpcmock.CheckPool(t, "kelp-rseth-tracker-pin-"+name, updated)
				ethrpcmock.CheckRequests(t, "kelp-rseth-tracker-pin-"+name, recorder)
			})
		})
	}
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     *PoolSimulator
		param             poolpkg.CalcAmountOutParams
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			// tx: 0x1509967c8aa688eb70d1c8d73a8cb71221e4f6b0a0648c7f2a7fd22062548236
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{
							"0xa1290d69c65a6fe4df752f95823fae25cb99e5a7", // rsETH
							"0xa35b1b31ce002fbf2058d22f30f95d405200a15b", // ETHx
						},
					},
				},

				minAmountToDeposit:  bignumber.NewBig("100000000000000"),
				totalDepositByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("802460400000000000000")},
				depositLimitByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("4197539600000000000000")},
				priceByAsset:        map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("1015786347348446492")},
				rsETHPrice:          bignumber.NewBig("1000000000000000000"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("2756400000000000000"),
					Token:  "0xa35b1b31ce002fbf2058d22f30f95d405200a15b",
				},
				TokenOut: "0xa1290d69c65a6fe4df752f95823fae25cb99e5a7",
			},
			expectedAmountOut: bignumber.NewBig("2799913487831257910"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}
