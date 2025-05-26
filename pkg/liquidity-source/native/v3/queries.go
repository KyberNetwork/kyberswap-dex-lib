package v3

import (
	"bytes"
	"math/big"
	"text/template"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
)

type (
	PoolsListQueryParams = uniswapv3.PoolsListQueryParams
	PoolTicksQueryParams = uniswapv3.PoolTicksQueryParams
)

func getPoolsListQuery(allowSubgraphError bool, lastCreatedAtTimestamp *big.Int, first, skip int) string {
	var tpl bytes.Buffer
	td := PoolsListQueryParams{
		AllowSubgraphError:     allowSubgraphError,
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
		First:                  first,
		Skip:                   skip,
	}

	// Add subgraphError: allow
	t, err := template.New("poolsListQuery").Parse(`{
		pools(
			{{ if .AllowSubgraphError }}subgraphError: allow,{{ end }}
			where: {
				createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }}
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
			feeTier
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

func getPoolTicksQuery(allowSubgraphError bool, poolAddress string, lastTickIdx string) string {
	var tpl bytes.Buffer
	td := PoolTicksQueryParams{
		AllowSubgraphError: allowSubgraphError,
		PoolAddress:        poolAddress,
		LastTickIdx:        lastTickIdx,
	}

	t, err := template.New("poolTicksQuery").Parse(`{
		ticks(
			{{ if .AllowSubgraphError }}subgraphError: allow,{{ end }}
			where: {
				pool: "{{.PoolAddress}}"
				{{ if .LastTickIdx }}tickIdx_gt: {{.LastTickIdx}},{{ end }}
				liquidityGross_not: 0
			},
			orderBy: tickIdx,
			orderDirection: asc,
			first: 1000
		) {
			tickIdx
			liquidityNet
			liquidityGross
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
