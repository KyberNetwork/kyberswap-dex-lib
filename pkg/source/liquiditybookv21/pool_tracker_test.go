package liquiditybookv21

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestGetNewPoolState(t *testing.T) {
	poolBytes := []byte(`{"address":"0xf4866fd96af220a82d38f29de5e638b231af143b","exchange":"liquiditybook-v21","type":"liquiditybook-v21","timestamp":1698122370,"reserves":["0","0"],"tokens":[{"address":"0xb8b3faa3256e65438b8459e838c598121ff6f1bc","weight":50,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","weight":50,"swappable":true}]}`)
	var e entity.Pool
	if err := json.Unmarshal(poolBytes, &e); err != nil {
		t.Fatal(err)
	}

	poolTracker, _ := NewPoolTracker(&Config{
		SubgraphAPI: "https://api.thegraph.com/subgraphs/name/vuquang23/traderjoe-v21-arbitrum",
	}, nil)

	_, _ = poolTracker.GetNewPoolState(context.Background(), e, pool.GetNewPoolStateParams{})

}
