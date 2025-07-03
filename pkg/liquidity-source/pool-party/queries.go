package poolparty

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

type PoolStateQueryParams struct {
	PoolAddress string
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
				createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }}
				poolType: "3"
				poolStatus: "ACTIVE"
				{{ .LastPoolIdsQuery }}
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
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

func getPoolState(poolAddress string) string {
	var tpl bytes.Buffer
	td := PoolStateQueryParams{
		poolAddress,
	}

	t, err := template.New("poolTicksQuery").Parse(`{
		pool(id: "{{ .PoolAddress }}") {
			id
			isVisible
			poolStatus
			publicAmountAvailable
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
