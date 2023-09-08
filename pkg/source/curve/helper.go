package curve

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

func initConfig(config *Config, ethrpcClient *ethrpc.Client) error {
	var (
		mainRegistryAddress, metaFactoryAddress, cryptoRegistryAddress, cryptoFactoryAddress common.Address
	)
	calls := ethrpcClient.NewRequest()

	calls.AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []interface{}{big.NewInt(0)},
	}, []interface{}{&mainRegistryAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []interface{}{big.NewInt(3)},
	}, []interface{}{&metaFactoryAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []interface{}{big.NewInt(5)},
	}, []interface{}{&cryptoRegistryAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []interface{}{big.NewInt(6)},
	}, []interface{}{&cryptoFactoryAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"addressProvider": config.AddressProvider,
			"error":           err,
		}).Errorf("failed to get address from address provider")
		return err
	}

	config.MainRegistryAddress = mainRegistryAddress.Hex()
	config.MetaPoolsFactoryAddress = metaFactoryAddress.Hex()
	config.CryptoPoolsRegistryAddress = cryptoRegistryAddress.Hex()
	config.CryptoPoolsFactoryAddress = cryptoFactoryAddress.Hex()

	return nil
}

func getAPrecisions(aList, aPreciseList []*big.Int) ([]*big.Int, error) {
	var aPrecisions = make([]*big.Int, len(aList))
	for i := 0; i < len(aPrecisions); i++ {
		if aList[i] != nil && aPreciseList[i] != nil {
			aPrecisions[i] = new(big.Int).Div(aPreciseList[i], aList[i])
		} else if aList[i] != nil {
			aPrecisions[i] = big.NewInt(1)
		} else {
			return nil, errors.New("A data did not get")
		}
	}

	return aPrecisions, nil
}

// extractNonZeroAddressesToStrings only uses for curve coin addresses.
// With the head of the array are a list of coin addresses, the tail of array are a list of addressZero
func extractNonZeroAddressesToStrings(addresses [8]common.Address) []string {
	var s []string
	for _, address := range addresses {
		if strings.EqualFold(address.Hex(), addressZero) {
			break
		}
		s = append(s, strings.ToLower(address.Hex()))
	}
	return s
}

func convertToEtherAddress(address string, chain int) string {
	if strings.EqualFold(strings.ToLower(address), addressEther) {
		return strings.ToLower(weth9[chain])
	}

	return address
}

func safeCastBigIntToString(num *big.Int) string {
	if num == nil {
		return emptyString
	}

	return num.String()
}

func safeCastBigIntToInt64(num *big.Int) int64 {
	if num == nil {
		return zero
	}

	return num.Int64()
}

func safeCastBigIntToReserve(num *big.Int) string {
	if num == nil {
		return zeroString
	}

	return num.String()
}
