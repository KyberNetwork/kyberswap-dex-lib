package pearl

import "encoding/json"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = json.Unmarshal([]byte(s), &staticExtra)

	return
}
