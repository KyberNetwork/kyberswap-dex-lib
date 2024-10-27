package velocimeter

import "github.com/bytedance/sonic"

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = sonic.Unmarshal([]byte(s), &staticExtra)

	return
}
