package zkerafinance

import (
	"math/big"
	"math/rand"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

func randomBigInt() *big.Int {
	words := make([]big.Word, 4)
	for i := range words {
		words[i] = big.Word(rand.Uint64())
	}
	return new(big.Int).SetBits(words)
}

func randomBool() bool { return rand.Int()%2 == 0 }

func randomAddress() common.Address {
	buf := make([]byte, common.AddressLength)
	for i := range buf {
		buf[i] = byte(rand.Uint64() % 256)
	}
	return common.BytesToAddress(buf)
}

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p := &PoolSimulator{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address:    randomAddress().Hex(),
					ReserveUsd: rand.Float64(),
					SwapFee:    randomBigInt(),
					Exchange:   DexType,
					Type:       DexType,
					Tokens: []string{
						randomAddress().Hex(),
						randomAddress().Hex(),
					},
					Reserves: []*big.Int{
						randomBigInt(),
						randomBigInt(),
					},
					Checked:     randomBool(),
					BlockNumber: rand.Uint64(),
				},
			},
			vault: &Vault{
				HasDynamicFees:           randomBool(),
				IncludeAmmPrice:          randomBool(),
				IsSwapEnabled:            randomBool(),
				StableSwapFeeBasisPoints: randomBigInt(),
				StableTaxBasisPoints:     randomBigInt(),
				SwapFeeBasisPoints:       randomBigInt(),
				TaxBasisPoints:           randomBigInt(),
				TotalTokenWeights:        randomBigInt(),

				WhitelistedTokens: []string{
					randomAddress().Hex(),
					randomAddress().Hex(),
				},
				PoolAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				BufferAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				ReservedAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				TokenDecimals: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				StableTokens: map[string]bool{
					randomAddress().Hex(): randomBool(),
					randomAddress().Hex(): randomBool(),
				},
				USDGAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				MaxUSDGAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				TokenWeights: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},

				PriceFeedAddress: randomAddress(),
				PriceFeed: &VaultPriceFeed{
					BNB:                        randomAddress().Hex(),
					BTC:                        randomAddress().Hex(),
					ETH:                        randomAddress().Hex(),
					FavorPrimaryPrice:          randomBool(),
					IsAmmEnabled:               randomBool(),
					IsSecondaryPriceEnabled:    randomBool(),
					MaxStrictPriceDeviation:    randomBigInt(),
					PriceSampleSpace:           randomBigInt(),
					SpreadThresholdBasisPoints: randomBigInt(),
					UseV2Pricing:               randomBool(),

					PriceDecimals: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					SpreadBasisPoints: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					AdjustmentBasisPoints: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					StrictStableTokens: map[string]bool{
						randomAddress().Hex(): randomBool(),
						randomAddress().Hex(): randomBool(),
					},
					IsAdjustmentAdditive: map[string]bool{
						randomAddress().Hex(): randomBool(),
						randomAddress().Hex(): randomBool(),
					},

					BNBBUSDAddress: randomAddress(),
					BNBBUSD: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					BTCBNBAddress: randomAddress(),
					BTCBNB: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					ETHBNBAddress: randomAddress(),
					ETHBNB: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					SecondaryPriceFeedAddress: randomAddress(),
					SecondaryPriceFeed: &FastPriceFeedV1{
						DisableFastPriceVoteCount: randomBigInt(),
						IsSpreadEnabled:           randomBool(),
						LastUpdatedAt:             randomBigInt(),
						MaxDeviationBasisPoints:   randomBigInt(),
						MinAuthorizations:         randomBigInt(),
						PriceDuration:             randomBigInt(),
						VolBasisPoints:            randomBigInt(),
						Prices: map[string]*big.Int{
							randomAddress().Hex(): randomBigInt(),
							randomAddress().Hex(): randomBigInt(),
						},
					},
					SecondaryPriceFeedVersion: rand.Int(),

					PriceFeedsAddresses: map[string]common.Address{
						randomAddress().Hex(): randomAddress(),
						randomAddress().Hex(): randomAddress(),
					},
					PriceFeeds: map[string]*PriceFeed{
						randomAddress().Hex(): {
							LatestAnswers: map[string]*big.Int{
								randomAddress().Hex(): randomBigInt(),
								randomAddress().Hex(): randomBigInt(),
							},
						},
						randomAddress().Hex(): {
							LatestAnswers: map[string]*big.Int{
								randomAddress().Hex(): randomBigInt(),
								randomAddress().Hex(): randomBigInt(),
							},
						},
					},
				},

				USDGAddress: randomAddress(),
				USDG: &USDG{
					Address:     randomAddress().Hex(),
					TotalSupply: randomBigInt(),
				},

				WhitelistedTokensCount: randomBigInt(),

				UseSwapPricing: randomBool(),
			},
			gas: Gas{
				Swap: rand.Int63(),
			},
		}
		p.vaultUtils = NewVaultUtils(p.vault)
		pools = append(pools, p)
	}
	{
		p := &PoolSimulator{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address:    randomAddress().Hex(),
					ReserveUsd: rand.Float64(),
					SwapFee:    randomBigInt(),
					Exchange:   DexType,
					Type:       DexType,
					Tokens: []string{
						randomAddress().Hex(),
						randomAddress().Hex(),
					},
					Reserves: []*big.Int{
						randomBigInt(),
						randomBigInt(),
					},
					Checked:     randomBool(),
					BlockNumber: rand.Uint64(),
				},
			},
			vault: &Vault{
				HasDynamicFees:           randomBool(),
				IncludeAmmPrice:          randomBool(),
				IsSwapEnabled:            randomBool(),
				StableSwapFeeBasisPoints: randomBigInt(),
				StableTaxBasisPoints:     randomBigInt(),
				SwapFeeBasisPoints:       randomBigInt(),
				TaxBasisPoints:           randomBigInt(),
				TotalTokenWeights:        randomBigInt(),

				WhitelistedTokens: []string{
					randomAddress().Hex(),
					randomAddress().Hex(),
				},
				PoolAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				BufferAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				ReservedAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				TokenDecimals: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				StableTokens: map[string]bool{
					randomAddress().Hex(): randomBool(),
					randomAddress().Hex(): randomBool(),
				},
				USDGAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				MaxUSDGAmounts: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},
				TokenWeights: map[string]*big.Int{
					randomAddress().Hex(): randomBigInt(),
					randomAddress().Hex(): randomBigInt(),
				},

				PriceFeedAddress: randomAddress(),
				PriceFeed: &VaultPriceFeed{
					BNB:                        randomAddress().Hex(),
					BTC:                        randomAddress().Hex(),
					ETH:                        randomAddress().Hex(),
					FavorPrimaryPrice:          randomBool(),
					IsAmmEnabled:               randomBool(),
					IsSecondaryPriceEnabled:    randomBool(),
					MaxStrictPriceDeviation:    randomBigInt(),
					PriceSampleSpace:           randomBigInt(),
					SpreadThresholdBasisPoints: randomBigInt(),
					UseV2Pricing:               randomBool(),

					PriceDecimals: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					SpreadBasisPoints: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					AdjustmentBasisPoints: map[string]*big.Int{
						randomAddress().Hex(): randomBigInt(),
						randomAddress().Hex(): randomBigInt(),
					},
					StrictStableTokens: map[string]bool{
						randomAddress().Hex(): randomBool(),
						randomAddress().Hex(): randomBool(),
					},
					IsAdjustmentAdditive: map[string]bool{
						randomAddress().Hex(): randomBool(),
						randomAddress().Hex(): randomBool(),
					},

					BNBBUSDAddress: randomAddress(),
					BNBBUSD: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					BTCBNBAddress: randomAddress(),
					BTCBNB: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					ETHBNBAddress: randomAddress(),
					ETHBNB: &PancakePair{
						Reserves: []*big.Int{
							randomBigInt(),
							randomBigInt(),
						},
						TimestampLast: rand.Uint32(),
					},

					SecondaryPriceFeedAddress: randomAddress(),
					SecondaryPriceFeed: &FastPriceFeedV2{
						DisableFastPriceVoteCount:     randomBigInt(),
						IsSpreadEnabled:               randomBool(),
						LastUpdatedAt:                 randomBigInt(),
						MaxDeviationBasisPoints:       randomBigInt(),
						MinAuthorizations:             randomBigInt(),
						PriceDuration:                 randomBigInt(),
						MaxPriceUpdateDelay:           randomBigInt(),
						SpreadBasisPointsIfChainError: randomBigInt(),
						SpreadBasisPointsIfInactive:   randomBigInt(),
						Prices: map[string]*big.Int{
							randomAddress().Hex(): randomBigInt(),
							randomAddress().Hex(): randomBigInt(),
						},
						PriceData: map[string]PriceDataItem{
							randomAddress().Hex(): {
								RefPrice:            randomBigInt(),
								RefTime:             rand.Uint64(),
								CumulativeRefDelta:  rand.Uint64(),
								CumulativeFastDelta: rand.Uint64(),
							},
							randomAddress().Hex(): {
								RefPrice:            randomBigInt(),
								RefTime:             rand.Uint64(),
								CumulativeRefDelta:  rand.Uint64(),
								CumulativeFastDelta: rand.Uint64(),
							},
						},
						MaxCumulativeDeltaDiffs: map[string]*big.Int{
							randomAddress().Hex(): randomBigInt(),
							randomAddress().Hex(): randomBigInt(),
						},
					},
					SecondaryPriceFeedVersion: rand.Int(),

					PriceFeedsAddresses: map[string]common.Address{
						randomAddress().Hex(): randomAddress(),
						randomAddress().Hex(): randomAddress(),
					},
					PriceFeeds: map[string]*PriceFeed{
						randomAddress().Hex(): {
							LatestAnswers: map[string]*big.Int{
								randomAddress().Hex(): randomBigInt(),
								randomAddress().Hex(): randomBigInt(),
							},
						},
						randomAddress().Hex(): {
							LatestAnswers: map[string]*big.Int{
								randomAddress().Hex(): randomBigInt(),
								randomAddress().Hex(): randomBigInt(),
							},
						},
					},
				},

				USDGAddress: randomAddress(),
				USDG: &USDG{
					Address:     randomAddress().Hex(),
					TotalSupply: randomBigInt(),
				},

				WhitelistedTokensCount: randomBigInt(),

				UseSwapPricing: randomBool(),
			},
			gas: Gas{
				Swap: rand.Int63(),
			},
		}
		p.vaultUtils = NewVaultUtils(p.vault)
		pools = append(pools, p)
	}
	return pools
}
