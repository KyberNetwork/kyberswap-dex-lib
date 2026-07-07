package poolparty

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListQueryParams struct {
	ChainID                int
	LastCreatedAtTimestamp int
	LastPoolIdsQuery       string
	First                  int
	Skip                   int
}

type PoolStateQueryParams struct {
	PoolAddress string
}

// getPoolsListQuery returns the GraphQL query string to get the list of pools.
// Pool-party update to different on-chain indexer, so the query is different
// with other subgraph queries.
func getPoolsListQuery(chainID valueobject.ChainID, lastCreatedAtTimestamp int, lastPoolIds []string) string {
	var tpl bytes.Buffer
	var lastPoolIdsQ string
	if len(lastPoolIds) > 0 {
		lastPoolIdsQ = fmt.Sprintf(", id: { _nin: [\"%s\"] }", strings.Join(lastPoolIds, "\",\""))
	} else {
		lastPoolIdsQ = ""
	}

	td := PoolsListQueryParams{
		int(chainID),
		lastCreatedAtTimestamp,
		lastPoolIdsQ,
		graphFirstLimit,
		0,
	}

	t, err := template.New("poolsListQuery").Parse(`{
		Pool(
			where: {
				chainID: { _eq: {{ .ChainID }} }
				createdAtTimestamp: { _gte: {{ .LastCreatedAtTimestamp }} }
				poolType: { _eq: "3" }
				poolStatus: { _eq: "ACTIVE" }
				{{ .LastPoolIdsQuery }}
			},
			limit: {{ .First }},
			offset: {{ .Skip }},
			order_by: {
				createdAtTimestamp: asc
			}
		) {
			id
			poolAddress
			tokenAddress
			tokenSymbol
			tokenDecimals
			isVisible
			poolStatus
			publicAmountAvailable
			createdAtTimestamp
		}
	}`)

	if err != nil {
		panic(err)
	}

	err = t.Execute(&tpl, td)

	if err != nil {
		panic(err)
	}

	return tpl.String()
}
