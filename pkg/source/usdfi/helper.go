package usdfi

import "github.com/goccy/go-json"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = json.Unmarshal([]byte(s), &staticExtra)

	return
}

func extractExtra(s string) (extra Extra, err error) {
	err = json.Unmarshal([]byte(s), &extra)

	return
}
