package bancorv21

import (
	"context"
	"embed"
	"fmt"
	"math/big"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
)

//go:embed sample_pool_data.txt
var sampleFile embed.FS

func TestGetNewPoolStatePinsRegistrySnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := common.HexToAddress("0x1000000000000000000000000000000000000001")
		anchor := common.HexToAddress("0x2000000000000000000000000000000000000002")
		converter := common.HexToAddress("0x3000000000000000000000000000000000000003")
		tokenA := common.HexToAddress("0x4000000000000000000000000000000000000004")
		tokenB := common.HexToAddress("0x5000000000000000000000000000000000000005")
		func() {
			srv, recorder := newBancorTrackerServer(t, 1, registry, anchor, converter, tokenA, tokenB)
			client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x6000000000000000000000000000000000000006"))
			tracker, err := NewPoolTracker(&Config{DexID: DexType, ConverterRegistry: registry.Hex()}, client)
			require.NoError(t, err)
			initial := entity.Pool{Address: "0x7000000000000000000000000000000000000007", Exchange: DexType, Type: DexType, Extra: "{}"}
			updated, err := tracker.GetNewPoolState(context.Background(), initial, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			if os.Getenv("BLOCKPIN_BASELINE") != "1" {
				require.Equal(t, uint64(ethrpcmock.Block), updated.BlockNumber)
				require.Equal(t, "aggregate", recorder.Snapshot()[0].OuterMethod)
			}
			require.Len(t, updated.Tokens, 0)
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
			require.Len(t, extra.InnerPools, 1)
			inner := extra.InnerPools[0]
			require.Equal(t, []string{hexutil.Encode(tokenA[:]), hexutil.Encode(tokenB[:])}, []string{inner.Tokens[0].Address, inner.Tokens[1].Address})
			require.Equal(t, entity.PoolReserves{"101", "202"}, inner.Reserves)
			require.Equal(t, float64(25), inner.SwapFee)
			var innerExtra ExtraInner
			require.NoError(t, json.Unmarshal([]byte(inner.Extra), &innerExtra))
			require.Equal(t, uint64(25), innerExtra.ConversionFee)
			require.Equal(t, []string{hexutil.Encode(anchor[:])}, extra.AnchorsByConvertibleToken[hexutil.Encode(tokenA[:])])
			require.Equal(t, []string{hexutil.Encode(tokenA[:]), hexutil.Encode(tokenB[:])}, extra.TokensByLpAddress[hexutil.Encode(anchor[:])])
			ethrpcmock.CheckPool(t, "bancor-v21_registry_nonempty", updated)
			ethrpcmock.CheckRequests(t, "bancor-v21_registry_nonempty", recorder)
			ethrpcmock.RequireRequestsPinned(t, recorder, 8)
		}()
		func() {
			srv, recorder := newBancorTrackerServer(t, 0, registry, anchor, converter, tokenA, tokenB)
			client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x6000000000000000000000000000000000000006"))
			tracker, err := NewPoolTracker(&Config{DexID: DexType, ConverterRegistry: registry.Hex()}, client)
			require.NoError(t, err)
			updated, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: "0x7000000000000000000000000000000000000007", Extra: "{}"}, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
			require.Empty(t, extra.InnerPools)
			if os.Getenv("BLOCKPIN_BASELINE") != "1" {
				require.Equal(t, uint64(ethrpcmock.Block), updated.BlockNumber)
			}
			ethrpcmock.RequireRequestsPinned(t, recorder, 5)
		}()
		func() {
			srv, recorder := newBancorTrackerServer(t, 1, registry, anchor, converter, tokenA, tokenB)
			client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x6000000000000000000000000000000000000006"))
			tracker, err := NewPoolTracker(&Config{DexID: DexType, ConverterRegistry: registry.Hex()}, client)
			require.NoError(t, err)
			initial := entity.Pool{Address: "0x7000000000000000000000000000000000000007", BlockNumber: ethrpcmock.Block + 1, Extra: "original"}
			updated, err := tracker.GetNewPoolState(context.Background(), initial, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, initial, updated)
			ethrpcmock.RequireRequestsPinned(t, recorder, 6)
		}()
	})
}

func newBancorTrackerServer(t *testing.T, pairCount int64, registry, anchor, converter, tokenA, tokenB common.Address) (*httptest.Server, *ethrpcmock.Recorder) {
	resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
		switch method.Name {
		case getAnchorCount:
			return []any{big.NewInt(pairCount)}, nil
		case registryGetAnchors:
			if pairCount == 0 {
				return []any{[]common.Address{}}, nil
			}
			return []any{[]common.Address{anchor}}, nil
		case getConvertersByAnchors:
			if pairCount == 0 {
				return []any{[]common.Address{}}, nil
			}
			return []any{[]common.Address{converter}}, nil
		case getConvertibleTokens:
			return []any{[]common.Address{tokenA, tokenB}}, nil
		case getConvertibleTokenAnchors:
			return []any{[]common.Address{anchor}}, nil
		case converterGetTokenCount:
			return []any{uint16(2)}, nil
		case converterGetTokens:
			if inputs[0].(*big.Int).Uint64() == 0 {
				return []any{tokenA}, nil
			}
			return []any{tokenB}, nil
		case converterGetReserve:
			if inputs[0].(common.Address) == tokenA {
				return []any{big.NewInt(101)}, nil
			}
			return []any{big.NewInt(202)}, nil
		case converterGetFee:
			return []any{uint32(25)}, nil
		default:
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = ethrpcmock.Value(output.Type)
			}
			return values, nil
		}
	}
	return ethrpcmock.NewWithResolver(t, resolver, converterRegistryABI, converterABI)
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	// Read the embedded file
	data, err := sampleFile.ReadFile("sample_pool_data.txt")
	if err != nil {
		fmt.Println("Error reading embedded file:", err)
		return
	}

	// Unmarshal the JSON data into the Pool struct
	var pool entity.Pool
	err = json.Unmarshal(data, &pool)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	poolSim, err := NewPoolSimulator(pool)
	assert.Nil(t, err)

	t.Run("Test rateByPath success calculate", func(t *testing.T) {
		eth := strings.ToLower("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
		bnt := strings.ToLower("0x1F573D6Fb3F13d689FF844B4cE37794d79a7FF1C")
		anchor1 := strings.ToLower("0xb1CD6e4153B2a390Cf00A6556b0fC1458C4A5533")
		samplePath := []string{eth, anchor1, bnt}
		result, err := poolSim.rateByPath(samplePath, new(big.Int).SetUint64(1000))
		assert.Nil(t, err)
		assert.Equal(t, "3899370", result.String())
	})

	t.Run("Test rateByPath can't calc with 0 reserves of pools", func(t *testing.T) {
		samplePath := []string{
			strings.ToLower("0x4Fabb145d64652a948d72533023f6E7A623C7C53"),
			strings.ToLower("0x7b86306D72103Ccd5405DF9dBFf4B794C46EBbC9"),
			strings.ToLower("0x1F573D6Fb3F13d689FF844B4cE37794d79a7FF1C"),
			strings.ToLower("0xE6b31fB3f29fbde1b92794B0867A315Ff605A324"),
			strings.ToLower("0xB8c77482e45F1F44dE1745F52C74426C631bDD52")}
		_, err := poolSim.rateByPath(samplePath, new(big.Int).SetUint64(1000))
		assert.Equal(t, err, ErrInvalidReserve)
	})

	t.Run("Should find path successfully", func(t *testing.T) {
		path := poolSim.findPath(
			strings.ToLower("0x4Fabb145d64652a948d72533023f6E7A623C7C53"),
			strings.ToLower("0xB8c77482e45F1F44dE1745F52C74426C631bDD52"))
		assert.Nil(t, err)
		assert.Equal(t, path, []string{
			"0x4fabb145d64652a948d72533023f6e7a623c7c53",
			"0x7b86306d72103ccd5405df9dbff4b794c46ebbc9",
			"0x1f573d6fb3f13d689ff844b4ce37794d79a7ff1c",
			"0xe6b31fb3f29fbde1b92794b0867a315ff605a324",
			"0xb8c77482e45f1f44de1745f52c74426c631bdd52"})
		t.Log(path, err)
	})
}
