package equalizer

import "github.com/goccy/go-json"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = json.Unmarshal([]byte(s), &staticExtra)

	return
}
