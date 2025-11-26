package nadfun

import (
	"bytes"
	"math/big"
	"text/template"
)

const (
	graphFirstLimit = 1000
)

type CurvesQueryParams struct {
	LastBlockTimestamp *big.Int
	First              int
}

type SubgraphCurve struct {
	Token          string `json:"token"`
	BlockTimestamp string `json:"blockTimestamp"`
}

func getCurvesQuery(lastBlockTimestamp *big.Int, first int) string {
	var tpl bytes.Buffer
	td := CurvesQueryParams{
		lastBlockTimestamp,
		first,
	}

	t, err := template.New("curvesQuery").Parse(`{
		curves(
			where: {
				blockTimestamp_gte: {{ .LastBlockTimestamp }},
				isLocked: false,
				isGraduated: false
			},
			first: {{ .First }},
			orderBy: blockTimestamp,
			orderDirection: asc
		) {
			token
			blockTimestamp
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
