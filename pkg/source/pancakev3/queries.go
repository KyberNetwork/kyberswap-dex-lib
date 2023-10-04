package pancakev3

import (
	"bytes"
	"math/big"
	"text/template"
)

type PoolsListQueryParams struct {
	AllowSubgraphError     bool
	LastCreatedAtTimestamp *big.Int
	First                  int
	Skip                   int
}

type PoolTicksQueryParams struct {
	AllowSubgraphError bool
	PoolAddress        string
	Skip               int
}

func getPoolsListQuery(allowSubgraphError bool, lastCreatedAtTimestamp *big.Int, first, skip int) string {
	var tpl bytes.Buffer
	td := PoolsListQueryParams{
		allowSubgraphError,
		lastCreatedAtTimestamp,
		first,
		skip,
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
		_meta { block { timestamp }}
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
