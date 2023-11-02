package usdfi

import "encoding/json"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = json.Unmarshal([]byte(s), &staticExtra)

	return
}

func extractExtra(s string) (extra Extra, err error) {
	err = json.Unmarshal([]byte(s), &extra)

	return
}
