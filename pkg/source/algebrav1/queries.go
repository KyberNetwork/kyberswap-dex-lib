package algebrav1

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"text/template"
)

type PoolsListQueryParams struct {
	AllowSubgraphError     bool
	LastCreatedAtTimestamp *big.Int
	First                  int
	Skip                   int
	LastPoolIdsQuery       string
}

type PoolTicksQueryParams struct {
	AllowSubgraphError bool
	PoolAddress        string
	Skip               int
}

func getPoolsListQuery(allowSubgraphError bool, lastCreatedAtTimestamp *big.Int, lastPoolIds []string, first, skip int) string {
	var tpl bytes.Buffer
	var lastPoolIdsQ string
	if len(lastPoolIds) > 0 {
		lastPoolIdsQ = fmt.Sprintf(", id_not_in: [\"%s\"]", strings.Join(lastPoolIds, "\",\""))
	} else {
		lastPoolIdsQ = ""
	}
	td := PoolsListQueryParams{
		allowSubgraphError,
		lastCreatedAtTimestamp,
		first,
		skip,
		lastPoolIdsQ,
	}

	// Add subgraphError: allow
	t, err := template.New("poolsListQuery").Parse(`{
		pools(
			{{ if .AllowSubgraphError }}subgraphError: allow,{{ end }}
			where: {
				createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }}
				{{ .LastPoolIdsQuery }}
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			token0 {
				id
				name
				symbol
				decimals
			}
			token1 {
				id
				name
				symbol
				decimals
			}
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

func getPoolTicksQuery(allowSubgraphError bool, poolAddress string, skip int) string {
	var tpl bytes.Buffer
	td := PoolTicksQueryParams{
		allowSubgraphError,
		poolAddress,
		skip,
	}

	t, err := template.New("poolTicksQuery").Parse(`{
		pool(
			{{ if .AllowSubgraphError }}subgraphError: allow,{{ end }}
			id: "{{.PoolAddress}}"
		) {
			id
			ticks(orderBy: tickIdx, orderDirection: asc, first: 1000, skip: {{.Skip}}) {
				tickIdx
				liquidityNet
				liquidityGross
			}
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
