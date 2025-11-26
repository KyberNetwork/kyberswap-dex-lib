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
