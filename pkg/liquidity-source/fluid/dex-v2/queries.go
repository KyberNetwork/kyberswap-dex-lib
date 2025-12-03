package dexv2

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type PoolsListQueryParams struct {
	LastCreatedAtTimestamp int
	LastPoolIdsQuery       string
	First                  int
	Skip                   int
}

type PoolTicksQueryParams struct {
	PoolId      string
	LastTickIdx int
}

func getPoolsListQuery(lastCreatedAtTimestamp int, lastPoolIds []string) string {
	var tpl bytes.Buffer
	var lastPoolIdsQ string
	if len(lastPoolIds) > 0 {
		lastPoolIdsQ = fmt.Sprintf(", id_not_in: [\"%s\"]", strings.Join(lastPoolIds, "\",\""))
	} else {
		lastPoolIdsQ = ""
	}

	td := PoolsListQueryParams{
		lastCreatedAtTimestamp,
		lastPoolIdsQ,
		graphFirstLimit,
		0,
	}

	t, err := template.New("poolsListQuery").Parse(`{
		pools(
			where: {
				createdAt_gte: {{ .LastCreatedAtTimestamp }}
				{{ .LastPoolIdsQuery }}
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: createdAt,
			orderDirection: asc
		) {
			id
			dexId
			dexType
			token0
			token1
			fee
			tickSpacing
			controller
			createdAt
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

func getPoolTicksQuery(poolId string, lastTickIdx int) string {
	var tpl bytes.Buffer
	td := PoolTicksQueryParams{
		poolId,
		lastTickIdx,
	}

	t, err := template.New("poolTicksQuery").Parse(`{
		ticks(
			where: {
				poolId: "{{.PoolId}}"
				{{ if .LastTickIdx }}tick_gt: {{.LastTickIdx}},{{ end }}
				liquidityGross_not: 0
			},
			orderBy: tick,
			orderDirection: asc,
			first: 1000
		) {
			tick
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
