package curve

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI           abi.ABI
	addressProviderABI abi.ABI
	mainRegistryABI    abi.ABI
	metaPoolFactoryABI abi.ABI
	cryptoFactoryABI   abi.ABI
	cryptoRegistryABI  abi.ABI
	metaABI            abi.ABI
	aaveABI            abi.ABI
	plainOracleABI     abi.ABI
	baseABI            abi.ABI
	twoABI             abi.ABI
	twoV2ABI           abi.ABI
	tricryptoABI       abi.ABI
	tricryptoV2ABI     abi.ABI
	oracleABI          abi.ABI
	compoundABI        abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIBytes},
		{&addressProviderABI, addressProviderABIBytes},
		{&mainRegistryABI, mainRegistryABIBytes},
		{&metaPoolFactoryABI, metaPoolFactoryABIBybtes},
		{&cryptoFactoryABI, cryptoFactoryABIBytes},
		{&cryptoRegistryABI, cryptoRegistryABIBytes},
		{&metaABI, metaABIBytes},
		{&aaveABI, aaveABIBytes},
		{&plainOracleABI, plainOraclePoolABIBytes},
		{&baseABI, basePoolABIBytes},
		{&twoABI, twoABIBytes},
		{&twoV2ABI, twoV2ABIBytes},
		{&tricryptoABI, tricryptoABIBytes},
		{&tricryptoV2ABI, tricryptoV2ABIBytes},
		{&oracleABI, oracleABIBytes},
		{&compoundABI, compoundABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
