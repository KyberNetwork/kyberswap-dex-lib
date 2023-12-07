package stable

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulator(t *testing.T) {
	poolJson := `{"address":"0x1d312eedd57e8d43bcb6369e4b8f02d3c18aaf13","exchange":"velocore-v2-stable","type":"velocore-v2-stable","timestamp":1699599200,"reserves":["13849396905","51671566387","55262649838"],"tokens":[{"address":"0x176211869ca2b568f2a7d4ee941e073a821ee1ff","weight":1,"swappable":true},{"address":"0x3f006b0493ff32b33be2809367f5f6722cb84a7b","weight":1,"swappable":true},{"address":"0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e","weight":1,"swappable":true}],"extra":"{\"amp\":1250000000000000,\"fee1e18\":100000000000000,\"lpTokenBalances\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":340282366920938463463374607418098781776,\"0x3f006b0493ff32b33be2809367f5f6722cb84a7b\":340282366920938463463374607380900924107,\"0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e\":340282366920938463463374607375521356737},\"tokenInfo\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":{\"indexPlus1\":1,\"scale\":12},\"0x3f006b0493ff32b33be2809367f5f6722cb84a7b\":{\"indexPlus1\":2,\"scale\":12},\"0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e\":{\"indexPlus1\":3,\"scale\":12}}}"}`
	poolEntity := entity.Pool{}
	err := json.Unmarshal([]byte(poolJson), &poolEntity)
	assert.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	assert.Nil(t, err)
	result, err := p.CalcAmountOut(
		pool.TokenAmount{
			Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			Amount: big.NewInt(3849396905),
		},
		"0x3f006b0493ff32b33be2809367f5f6722cb84a7b")
	assert.Nil(t, err)
	t.Error(result.TokenAmountOut.Amount.String())
}
