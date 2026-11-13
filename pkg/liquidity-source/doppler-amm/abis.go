package doppleramm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ammABI    abi.ABI
	quoterABI abi.ABI
	launchABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ammABI, ammABIData},
		{&quoterABI, quoterABIData},
		{&launchABI, launchModuleABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}

	// The module is unverified, so two getters are known by selector only (found
	// by probing the deployed bytecode); pin their IDs under placeholder names.
	pinSelector(&launchABI, "levelBounds", [4]byte{0x5c, 0xd8, 0x7b, 0xa7})
	pinSelector(&launchABI, "levelInfo", [4]byte{0x56, 0x49, 0x5c, 0x7c})
}

func pinSelector(a *abi.ABI, name string, id [4]byte) {
	m := a.Methods[name]
	m.ID = id[:]
	a.Methods[name] = m
}
