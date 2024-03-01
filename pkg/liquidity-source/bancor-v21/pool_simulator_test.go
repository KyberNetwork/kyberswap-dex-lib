package bancorv21

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

//go:embed sample_pool_data.txt
var sampleFile embed.FS

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
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
		assert.Equal(t, result.String(), "3898108")
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
		t.Log(path, err)
	})
}
