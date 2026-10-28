package ethervista

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
)

var (
	MaxUInt256 = new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 256), uint256.NewInt(1))
)

func TestSafeAdd(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(15),
		},
		{
			name: "overflow case",
			args: args{
				x: new(uint256.Int).Set(MaxUInt256),
				y: uint256.NewInt(10),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathAddOverflow.Error(), func() {
					SafeAdd(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeAdd(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeSub(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(5),
		},
		{
			name: "underflow case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(15),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathSubUnderflow.Error(), func() {
					SafeSub(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeSub(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeMul(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(50),
		},
		{
			name: "overflow case",
			args: args{
				x: new(uint256.Int).Set(MaxUInt256),
				y: uint256.NewInt(10),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathMulOverflow.Error(), func() {
					SafeMul(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeMul(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetNewPoolStatePinsRouterReadsToPairBlock(t *testing.T) {
	for _, withOverrides := range []bool{false, true} {
		name := "normal"
		if withOverrides {
			name = "overrides"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const pairAddress = "0x0000000000000000000000000000000000000001"
				const factoryAddress = "0x0000000000000000000000000000000000000002"
				const routerAddress = "0x0000000000000000000000000000000000000003"
				resolver := func(_ common.Address, method abi.Method, _ []any) ([]any, error) {
					switch method.RawName {
					case pairMethodGetReserves:
						return []any{big.NewInt(1000), big.NewInt(2000), uint32(1)}, nil
					case pairMethodBuyTotalFee, pairMethodSellTotalFee:
						return []any{uint8(2)}, nil
					case factoryMethodRouter:
						return []any{common.HexToAddress(routerAddress)}, nil
					case routerMethodUSDCToEth:
						return []any{big.NewInt(99)}, nil
					default:
						t.Fatalf("unexpected method %s", method.RawName)
						return nil, nil
					}
				}
				srv, recorder := ethrpcmock.NewWithResolver(t, resolver, pairABI, factoryABI, routerABI)
				client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x0000000000000000000000000000000000000005"))
				tracker, err := NewPoolTracker(&Config{DexID: DexType, FactoryAddress: factoryAddress}, client)
				require.NoError(t, err)
				p := entity.Pool{Address: pairAddress, BlockNumber: 1}
				var updated entity.Pool
				if withOverrides {
					updated, err = tracker.GetNewPoolStateWithOverrides(context.Background(), p, pool.GetNewPoolStateWithOverridesParams{
						Overrides: map[common.Address]gethclient.OverrideAccount{common.HexToAddress("0x0000000000000000000000000000000000000009"): {Balance: big.NewInt(1)}},
					})
				} else {
					updated, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
				}
				require.NoError(t, err)
				require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
				require.Equal(t, entity.PoolReserves{"1000", "2000"}, updated.Reserves)
				ethrpcmock.RequireRequestsPinned(t, recorder, 2)
				ethrpcmock.CheckPool(t, "ether-vista-tracker-pin", updated)
				key := "ether-vista-tracker-pin-normal"
				if withOverrides {
					key = "ether-vista-tracker-pin-overrides"
				}
				ethrpcmock.CheckRequests(t, key, recorder)
			})
		})
	}
}
