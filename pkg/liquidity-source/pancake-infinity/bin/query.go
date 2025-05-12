package bin

import (
	"bytes"
	"text/template"
)

type PoolsListQueryParams struct {
	LastCreatedAtTimestamp int
	First                  int
	Skip                   int
}

type BinsQueryParams struct {
	PoolAddress        string
	LastBinId          int32
	AllowSubgraphError bool
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
		lbpairs(
			where: {
				timestamp_gte: {{ .LastCreatedAtTimestamp }}
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: timestamp,
			orderDirection: asc
		) {
			id
			tokenX {
				id
				decimals
			}
			tokenY {
				id
				decimals
			}
			parameters
			hooks
			timestamp
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

func getBinsQuery(poolAddress string, lastBinId int32, allowSubgraphError bool) string {
	var tpl bytes.Buffer

	td := BinsQueryParams{
		PoolAddress:        poolAddress,
		LastBinId:          lastBinId,
		AllowSubgraphError: allowSubgraphError,
	}

	tmpl := `{
  lbpair(
    {{- if .AllowSubgraphError }}
    subgraphError: allow,
    {{- end }}
    id: "{{.PoolAddress}}"
  ) {
    tokenX { decimals }
    tokenY { decimals }
    reserveX
    reserveY
    bins(
      where: {
        binId_gt: {{.LastBinId}}
      },
      orderBy: binId,
      orderDirection: asc,
      first: 1000
    ) {
      binId
      reserveX
      reserveY
    }
  }
}
`

	t, err := template.New("binsQuery").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	if err := t.Execute(&tpl, td); err != nil {
		panic(err)
	}

	return tpl.String()
}
