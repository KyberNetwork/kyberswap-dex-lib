package capricornpamm

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	pammPoolABI       abi.ABI
	pricingEngineABI  abi.ABI
	oracleRegistryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pammPoolABI, pammPoolBytes},
		{&pricingEngineABI, pricingEngineBytes},
		{&oracleRegistryABI, oracleRegistryBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	overrideSelector(&oracleRegistryABI, methodGetPrice, "9da321aa")
	overrideSelector(&oracleRegistryABI, methodMaxPushPriceAge, "a3ac42e2")
	overrideSelector(&oracleRegistryABI, methodPythValidTimePeriod, "c1ee2358")
}

func overrideSelector(a *abi.ABI, name, hexID string) {
	m, ok := a.Methods[name]
	if !ok {
		panic(fmt.Sprintf("capricorn-pamm: ABI method %q missing for selector override", name))
	}

	m.ID = common.Hex2Bytes(hexID)
	a.Methods[name] = m
}
