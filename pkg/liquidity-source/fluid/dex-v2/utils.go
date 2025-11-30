package dexv2

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

func encodeFluidDexV2PoolAddress(dexId string, dexType int) string {
	return fmt.Sprintf("%s_d%d", dexId, dexType)
}

func parseFluidDexV2PoolAddress(address string) (string, int) {
	parts := strings.Split(address, "_d")
	dexType, _ := strconv.Atoi(parts[1])

	return parts[0], dexType
}

func calculateMappingStorageSlot(slot uint64, key common.Address) common.Hash {
	paddedKey := common.LeftPadBytes(key.Bytes(), 32)

	slotBig := new(big.Int).SetUint64(slot)
	paddedSlot := common.LeftPadBytes(slotBig.Bytes(), 32)

	input := append(paddedKey, paddedSlot...)

	return crypto.Keccak256Hash(input)
}

func calculateReservesFromTicks(sqrtPriceX96 *big.Int, ticks []Tick) (*big.Int, *big.Int, error) {
	L := big.NewInt(0)
	totalAmount0, totalAmount1 := big.NewInt(0), big.NewInt(0)

	for i, tickLower := range ticks {
		L.Add(L, tickLower.LiquidityNet)

		if L.Sign() == 0 {
			continue
		}

		if i == len(ticks)-1 {
			return nil, nil, errors.New("sum liquidity net is not zero")
		}

		tickUpper := ticks[i+1]

		sqrtLower, err := utils.GetSqrtRatioAtTick(tickLower.Index)
		if err != nil {
			return nil, nil, err
		}
		sqrtUpper, err := utils.GetSqrtRatioAtTick(tickUpper.Index)
		if err != nil {
			return nil, nil, err
		}

		var numer, denom, amount0, amount1, tmp big.Int
		if sqrtPriceX96.Cmp(sqrtLower) < 0 {
			numer.Mul(L, Q96).Mul(&numer, tmp.Sub(sqrtUpper, sqrtLower))
			denom.Mul(sqrtLower, sqrtUpper)

			amount0.Div(&numer, &denom)
		} else if sqrtPriceX96.Cmp(sqrtUpper) >= 0 {
			numer.Mul(L, tmp.Sub(sqrtUpper, sqrtLower))

			amount1.Div(&numer, Q96)
		} else {
			numer.
				Mul(L, Q96).
				Mul(&numer, tmp.Sub(sqrtUpper, sqrtPriceX96))
			denom.Mul(sqrtPriceX96, sqrtUpper)
			amount0.Div(&numer, &denom)

			numer.Mul(L, tmp.Sub(sqrtPriceX96, sqrtLower))
			amount1.Div(&numer, Q96)
		}

		totalAmount0.Add(totalAmount0, &amount0)
		totalAmount1.Add(totalAmount1, &amount1)
	}

	return totalAmount0, totalAmount1, nil
}
