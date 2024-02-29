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

	eth := strings.ToLower("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
	bnt := strings.ToLower("0x1F573D6Fb3F13d689FF844B4cE37794d79a7FF1C")
	anchor1 := strings.ToLower("0xb1CD6e4153B2a390Cf00A6556b0fC1458C4A5533")
	samplePath := []string{eth, anchor1, bnt}
	result, err := poolSim.rateByPath(samplePath, new(big.Int).SetUint64(1000))
	assert.Nil(t, err)
	t.Log(result.String())
}
