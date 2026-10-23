package ghost

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

func TestGetNewPoolState_InvalidStaticExtra(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{DexID: DexType}, nil)

	_, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address:     "ghost_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
		StaticExtra: "not-json",
	}, pool.GetNewPoolStateParams{})

	assert.Error(t, err)
}

func TestGetNewPoolStateSnapshotEquivalent(t *testing.T) {
	for _, resolution := range []string{"linear", "routing_direct", "routing_fallback"} {
		t.Run(resolution, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const sourceZero = "0x0000000000000000000000000000000000000001"
				const sourceOne = "0x0000000000000000000000000000000000000002"
				const targetZero = "0x0000000000000000000000000000000000000003"
				const targetOne = "0x0000000000000000000000000000000000000004"
				const recipientZero = "0x0000000000000000000000000000000000000005"
				const recipientOne = "0x0000000000000000000000000000000000000006"
				const directZero = "0x0000000000000000000000000000000000000007"
				const directOne = "0x0000000000000000000000000000000000000008"
				const fallbackZero = "0x0000000000000000000000000000000000000009"
				const fallbackOne = "0x000000000000000000000000000000000000000a"
				const tokenZero = "0x000000000000000000000000000000000000000b"
				const tokenOne = "0x000000000000000000000000000000000000000c"
				recipientByRouter := map[common.Address]common.Address{
					common.HexToAddress(sourceZero): common.HexToAddress(recipientZero),
					common.HexToAddress(sourceOne):  common.HexToAddress(recipientOne),
				}
				feeRecipientByRouter := map[common.Address]common.Address{
					common.HexToAddress(recipientZero): common.HexToAddress(directZero),
					common.HexToAddress(recipientOne):  common.HexToAddress(directOne),
				}
				fallbackByRecipient := map[common.Address]common.Address{
					common.HexToAddress(recipientZero): common.HexToAddress(fallbackZero),
					common.HexToAddress(recipientOne):  common.HexToAddress(fallbackOne),
				}
				resolver := func(target common.Address, method abi.Method, inputs []any) ([]any, error) {
					switch method.RawName {
					case "feeRecipient":
						return []any{recipientByRouter[target]}, nil
					case "feeType":
						if _, ok := recipientByRouter[target]; ok && resolution != "linear" {
							return []any{uint8(feeTypeCrossCollateralRouting)}, nil
						}
						return []any{uint8(feeTypeOffchainQuotedLinear)}, nil
					case "feeContracts":
						recipient := target
						if resolution == "routing_fallback" && common.Hash(inputs[1].([32]uint8)) != defaultRouterKey {
							return []any{common.Address{}}, nil
						}
						if resolution == "routing_direct" {
							return []any{feeRecipientByRouter[recipient]}, nil
						}
						return []any{fallbackByRecipient[recipient]}, nil
					case "maxFee":
						return []any{big.NewInt(20)}, nil
					case "halfAmount":
						return []any{big.NewInt(1000)}, nil
					case "quotes":
						return []any{big.NewInt(10), big.NewInt(200), big.NewInt(1), big.NewInt(4102444800)}, nil
					case "balanceOf":
						if target == common.HexToAddress(tokenOne) {
							return []any{big.NewInt(111)}, nil
						}
						return []any{big.NewInt(222)}, nil
					}
					values := make([]any, len(method.Outputs))
					for i, output := range method.Outputs {
						values[i] = ethrpcmock.Value(output.Type)
					}
					return values, nil
				}
				server, recorder := ethrpcmock.NewWithResolver(t, resolver, routerABI, routingFeeABI, feeABI, utilabi.Erc20ABI)
				client := ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0x0000000000000000000000000000000000000010"))
				static := StaticExtra{
					ZeroToOne: DirectionStatic{SourceRouter: sourceZero, TargetRouter: targetZero, LocalDomain: 1, ScaleNumerator: "1", ScaleDenominator: "1"},
					OneToZero: DirectionStatic{SourceRouter: sourceOne, TargetRouter: targetOne, LocalDomain: 1, ScaleNumerator: "1", ScaleDenominator: "1"},
				}
				staticBytes, err := json.Marshal(static)
				require.NoError(t, err)
				tracker := NewPoolTracker(&Config{DexID: DexType}, client)
				updated, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
					Address: "ghost_pool", Exchange: DexType, Type: DexType, StaticExtra: string(staticBytes), Extra: "{}",
					Tokens: []*entity.PoolToken{{Address: tokenZero}, {Address: tokenOne}}, Reserves: entity.PoolReserves{"0", "0"},
				}, pool.GetNewPoolStateParams{})
				require.NoError(t, err)

				require.Equal(t, ethrpcmock.Block, updated.BlockNumber)

				require.Equal(t, entity.PoolReserves{"222", "111"}, updated.Reserves)
				ethrpcmock.CheckPool(t, "ghost_"+resolution, updated)
				ethrpcmock.CheckRequests(t, "ghost_"+resolution, recorder, [2]string{"call", "aggregate"})
				ethrpcmock.RequireRequestsPinned(t, recorder, 2)
			})
		})
	}
}
