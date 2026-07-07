package cloberob

import (
	"bytes"
	"text/template"
)

type BooksQueryParams struct {
	LastCreatedAtTimestamp int64
	First                  int
	Skip                   int
}

func getBooksQuery(lastCreatedAtTimestamp int64, first int) string {
	var tpl bytes.Buffer
	t := template.Must(template.New("BooksQuery").Parse(`{
		books (
			where: { createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }} },
			first: {{ .First }}
			skip: {{ .Skip }}
			orderBy: createdAtTimestamp
			orderDirection: asc
		) {
			id
			unitSize
			makerPolicy
			makerFee
			isMakerFeeInQuote
			takerPolicy
			takerFee
			isTakerFeeInQuote
			base {
				id
  				name
				symbol
				decimals
			}
			quote {
				id
				name
				symbol
				decimals
			}
			hooks
			tick
			createdAtTimestamp
		}
	}`))

	err := t.Execute(&tpl, BooksQueryParams{
		lastCreatedAtTimestamp,
		first,
		0,
	})

	if err != nil {
		panic(err)
	}

	return tpl.String()
}
