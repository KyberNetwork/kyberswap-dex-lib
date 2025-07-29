package uniswapv4

import (
	"bytes"
	"text/template"
)

type PoolsListQueryParams struct {
	LastCreatedAtTimestamp int
	First                  int
	Skip                   int
}

type PoolTicksQueryParams struct {
	AllowSubgraphError bool
	PoolAddress        string
	LastTickIdx        string
}

func getPoolsListQuery(lastCreatedAtTimestamp int, first int) string {
	var tpl bytes.Buffer
	td := PoolsListQueryParams{
		lastCreatedAtTimestamp,
		first,
		0,
	}

	// Add subgraphError: allow
	t, err := template.New("poolsListQuery").Parse(`{
		pools(
			where: {
				createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }}
				id_in: [
					"0x9c03436d98a853a141f730558f1cd0e7fa0e405a8e80e7fac92751ad97a0aed5",
					"0x8f484d0af27622d51f2fd103d8d279ac4990e0849579bcf1fcd3e63369a6eb49"
				]
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
			token0 {
				id
				decimals
			}
			token1 {
				id
				decimals
			}
			feeTier
			tickSpacing
			hooks
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
