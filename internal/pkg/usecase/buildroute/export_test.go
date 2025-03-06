package buildroute

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func NewUnsignedTransaction(sender string, recipient string, data string,
	value *big.Int, gasPrice *big.Int) UnsignedTransaction {
	return UnsignedTransaction{
		sender,
		data,
		value,
		nil,
	}
}

func ConvertTransactionToMsg(tx UnsignedTransaction, routerAddress string) ethereum.CallMsg {
	var (
		from           = common.HexToAddress(tx.sender)
		to             = common.HexToAddress(routerAddress)
		encodedData, _ = hexutil.Decode(tx.data)
	)
	return ethereum.CallMsg{
		From:       from,
		To:         &to,
		Gas:        0,
		GasPrice:   tx.gasPrice,
		GasFeeCap:  nil,
		GasTipCap:  nil,
		Value:      tx.value,
		Data:       encodedData,
		AccessList: nil,
	}
}

func TestBuildRouteUseCase_EstimateRFQSlippage(t *testing.T) {
	testCases := []struct {
		name              string
		routeSummary      valueobject.RouteSummary
		slippageTolerance int64
		config            Config
		err               error
	}{
		{
			name: "route summary must not be changed",
			routeSummary: valueobject.RouteSummary{
				TokenIn:   "0x5947bb275c521040051d82396192181b413227a3",
				AmountIn:  bignumber.NewBig("856037931697362767875"),
				TokenOut:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
				AmountOut: bignumber.NewBig("18786230990"),
				Route: [][]valueobject.Swap{
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("42801896584868138408"),
							AmountOut:  bignumber.NewBig("25692705627735329427"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("25692705627735329427"),
							AmountOut:  bignumber.NewBig("962153526"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("42801896584868138393"),
							AmountOut:  bignumber.NewBig("25666460188550926460"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("25666460188550926460"),
							AmountOut:  bignumber.NewBig("960948539"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("659149207406969331251"),
							AmountOut:  bignumber.NewBig("385088281340775062510"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("385088281340775062510"),
							AmountOut:  bignumber.NewBig("14416146205"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("68483034535789021430"),
							AmountOut:  bignumber.NewBig("40413581564512931352"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("40413581564512931352"),
							AmountOut:  bignumber.NewBig("1512789500"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("42801896584868138393"),
							AmountOut:  bignumber.NewBig("24956969761768883822"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("24956969761768883822"),
							AmountOut:  bignumber.NewBig("934193220"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
				},
			},
			slippageTolerance: 0,
			config: Config{
				RFQAcceptableSlippageFraction: 1000,
			},
			err: nil,
		},
		{
			name: "route summary must not be changed",
			routeSummary: valueobject.RouteSummary{
				TokenIn:   "0x5947bb275c521040051d82396192181b413227a3",
				AmountIn:  bignumber.NewBig("381"), // 1 + 17 + 11 + 83 + 269
				TokenOut:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
				AmountOut: bignumber.NewBig("590"), // 13 + 10 + 97 + 271 + 199
				Route: [][]valueobject.Swap{
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("1"),
							AmountOut:  bignumber.NewBig("3"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("3"),
							AmountOut:  bignumber.NewBig("13"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("13"),
							AmountOut:  bignumber.NewBig("31"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("31"),
							AmountOut:  bignumber.NewBig("10"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("10"),
							AmountOut:  bignumber.NewBig("7"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("7"),
							AmountOut:  bignumber.NewBig("97"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("97"),
							AmountOut:  bignumber.NewBig("79"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("79"),
							AmountOut:  bignumber.NewBig("271"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
					{
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("271"),
							AmountOut:  bignumber.NewBig("251"),
							Exchange:   "traderjoe",
							PoolType:   "uniswap-v2",
						},
						{
							Pool:       "0xabc",
							SwapAmount: bignumber.NewBig("251"),
							AmountOut:  bignumber.NewBig("199"),
							Exchange:   "uniswapv3",
							PoolType:   "uniswapv3",
						},
					},
				},
			},
			slippageTolerance: 0,
			config: Config{
				RFQAcceptableSlippageFraction: 1000,
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewBuildRouteUseCase(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				tc.config,
			)

			routeSummary, err := usecase.estimateRFQSlippage(context.Background(), tc.routeSummary, tc.slippageTolerance)
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}

			assert.Equal(t, tc.routeSummary, routeSummary)
		})
	}
}
