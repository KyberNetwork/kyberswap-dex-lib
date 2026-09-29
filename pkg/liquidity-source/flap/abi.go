package flap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// portalABI is a var initializer (not populated inside init()) so other package-level vars that
// depend on it (e.g. tokenCreatedEventTopic in pool_factory.go) are guaranteed to see it fully
// populated: Go orders var initializers by dependency, but a var read inside another package's/file's
// init() func has no such guarantee relative to sibling var initializers.
var portalABI = mustParseABI(portalABIBytes)

func mustParseABI(data []byte) abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return parsed
}
