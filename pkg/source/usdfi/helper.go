package usdfi

import "github.com/bytedance/sonic"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = sonic.Unmarshal([]byte(s), &staticExtra)

	return
}

func extractExtra(s string) (extra Extra, err error) {
	err = sonic.Unmarshal([]byte(s), &extra)

	return
}
