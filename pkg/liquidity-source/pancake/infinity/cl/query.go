package cl

import (
	"bytes"
	"fmt"
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
totalValueLockedUSD_gt: "100"
      hooks_not_in: ["0x1a3dfbcac585e22f993cc8e09bcc0db388cc1ca3", "0x1e9c64cad39ddd36fb808e004067cffc710eb71d", "0xf27b9134b23957d842b08ffa78b07722fb9845bd", "0x0fcf6d110cf96be56d251716e69e37619932edf2", "0xdfdfb2c5a717ab00b370e883021f20c2fbaed277", "0x32c59d556b16db81dfc32525efb3cb257f7e493d", "0x0000000000000000000000000000000000000000"]
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
			parameters
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

	fmt.Println(tpl.String())
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
