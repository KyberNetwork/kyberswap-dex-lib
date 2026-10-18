package honey

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func s(s string) *uint256.Int {
	res, _ := uint256.FromDecimal(s)
	return res
}

func TestGetNewPoolStatePinsNoAssetRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resolver := func(_ common.Address, method abi.Method, _ []any) ([]any, error) {
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = ethrpcmock.Value(output.Type)
			}
			switch method.Name {
			case "numRegisteredAssets":
				values[0] = big.NewInt(0)
			case "polFeeCollectorFeeRate":
				values[0] = big.NewInt(7)
			}
			return values, nil
		}
		srv, recorder := ethrpcmock.NewWithResolver(t, resolver, honeyABI, erc20ABI, assetVaultABI)
		client := ethrpc.New(srv.URL)
		updated, err := NewPoolTracker(&Config{}, client).GetNewPoolState(context.Background(), entity.Pool{
			Address: "0x0000000000000000000000000000000000000001",
			Extra:   "{}",
		}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
		require.Equal(t, uint64(7), extra.PolFeeCollectorFeeRate.Uint64())
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
		ethrpcmock.CheckPool(t, "honey-tracker-pin", updated)
		ethrpcmock.CheckRequests(t, "honey-tracker-pin", recorder)
	})
}

func TestGetNewPoolStatePinsNewAssetRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		assets := []common.Address{
			common.HexToAddress("0x0000000000000000000000000000000000000011"),
			common.HexToAddress("0x0000000000000000000000000000000000000012"),
		}
		vaults := []common.Address{
			common.HexToAddress("0x0000000000000000000000000000000000000021"),
			common.HexToAddress("0x0000000000000000000000000000000000000022"),
		}
		resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = ethrpcmock.Value(output.Type)
			}
			switch method.Name {
			case "numRegisteredAssets":
				values[0] = big.NewInt(2)
			case "registeredAssets":
				values[0] = assets[inputs[0].(*big.Int).Int64()]
			case "vaults":
				asset := inputs[0].(common.Address)
				for i := range assets {
					if assets[i] == asset {
						values[0] = vaults[i]
					}
				}
			case "polFeeCollectorFeeRate":
				values[0] = big.NewInt(7)
			case "decimals":
				values[0] = uint8(6)
			case "mintRates":
				values[0] = big.NewInt(2)
			case "redeemRates":
				values[0] = big.NewInt(3)
			case "maxRedeem":
				values[0] = big.NewInt(4)
			}
			return values, nil
		}
		srv, recorder := ethrpcmock.NewWithResolver(t, resolver, honeyABI, erc20ABI, assetVaultABI)
		client := ethrpc.New(srv.URL)
		updated, err := NewPoolTracker(&Config{}, client).GetNewPoolState(context.Background(), entity.Pool{
			Address: "0x0000000000000000000000000000000000000001",
			Extra:   "{}",
		}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
		require.Equal(t, uint64(7), extra.PolFeeCollectorFeeRate.Uint64())
		require.Equal(t, []string{assets[0].Hex(), assets[1].Hex()}, extra.RegisteredAssets)
		require.Equal(t, []string{vaults[0].Hex(), vaults[1].Hex()}, extra.Vaults)
		require.Equal(t, []uint8{6, 6}, extra.AssetsDecimals)
		require.Equal(t, []uint8{6, 6}, extra.VaultsDecimals)
		require.Equal(t, []string{"2", "2"}, []string{extra.MintRates[0].String(), extra.MintRates[1].String()})
		require.Equal(t, []string{"3", "3"}, []string{extra.RedeemRates[0].String(), extra.RedeemRates[1].String()})
		require.Len(t, updated.Tokens, 2)
		require.Len(t, updated.Reserves, 2)
		ethrpcmock.RequireRequestsPinned(t, recorder, 4)
		ethrpcmock.CheckPool(t, "honey-tracker-new-assets", updated)
		ethrpcmock.CheckRequests(t, "honey-tracker-new-assets", recorder)
	})
}

func TestGetAmountOut(t *testing.T) {
	// https://dashboard.tenderly.co/tien7668/project/simulator/8d6e4d71-ac1a-4ae3-a42d-d618a008d37d/debugger?trace=0.7.1.3.2.1.5.4
	p := &PoolSimulator{
		redeemRates:            []*uint256.Int{s("999500000000000000"), s("999500000000000000")},
		polFeeCollectorFeeRate: u256.BONE,
	}

	shares, _, polFeeCollectorFeeShares := p.getSharesRedeemedFromHoney(s("503373735599534552165958"), 1)
	assert.Equal(t, shares, s("503122048731734784889875"))
	assert.Equal(t, polFeeCollectorFeeShares, s("251686867799767276083"))
}
