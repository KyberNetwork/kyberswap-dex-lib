package uniswapv3

import (
	"bytes"
	"math/big"
	"text/template"
)

type PoolsListQueryParams struct {
	SubgraphPoolField      string
	IncludeFeeTier         bool
	AllowSubgraphError     bool
	LastCreatedAtTimestamp *big.Int
	First                  int
	Skip                   int
}

type PoolTicksQueryParams struct {
	AllowSubgraphError bool
	PoolAddress        string
	LastTickIdx        string
}

func getPoolsListQuery(subgraphPoolField string, includeFeeTier, allowSubgraphError bool,
	lastCreatedAtTimestamp *big.Int, first, skip int) string {
	var tpl bytes.Buffer
	if subgraphPoolField == "" {
		subgraphPoolField = "pools"
	}
	td := PoolsListQueryParams{
		SubgraphPoolField:      subgraphPoolField,
		IncludeFeeTier:         includeFeeTier,
		AllowSubgraphError:     allowSubgraphError,
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
		First:                  first,
		Skip:                   skip,
	}

	t, err := template.New("poolsListQuery").Parse(`{
		{{ .SubgraphPoolField }}(
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
			createdAtTimestamp
			{{ if .IncludeFeeTier }}feeTier{{ end }}
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
		allowSubgraphError,
		poolAddress,
		lastTickIdx,
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
		_meta { block { timestamp } }
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
