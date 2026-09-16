package everlongflamm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	flammABI      abi.ABI
	hookABI       abi.ABI
	levHookABI    abi.ABI
	spreadHookABI abi.ABI
	priceFeedABI  abi.ABI
	routerABI     abi.ABI
	accountABI    abi.ABI
	factoryABI    abi.ABI
	morphoABI     abi.ABI
	irmABI        abi.ABI
	aggregatorABI abi.ABI
	oracleABI     abi.ABI
	multicallABI  abi.ABI
)

func init() {
	for _, e := range []struct {
		target *abi.ABI
		data   []byte
	}{
		{&flammABI, flammABIJson},
		{&hookABI, hookABIJson},
		{&levHookABI, levHookABIJson},
		{&spreadHookABI, spreadHookABIJson},
		{&priceFeedABI, priceFeedABIJson},
		{&routerABI, routerABIJson},
		{&accountABI, accountABIJson},
		{&factoryABI, factoryABIJson},
		{&morphoABI, morphoABIJson},
		{&irmABI, irmABIJson},
		{&aggregatorABI, aggregatorABIJson},
		{&oracleABI, oracleABIJson},
		{&multicallABI, multicallABIJson},
	} {
		parsed, err := abi.JSON(bytes.NewReader(e.data))
		if err != nil {
			panic(err)
		}
		*e.target = parsed
	}
}
