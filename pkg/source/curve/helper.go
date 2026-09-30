package curve

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

func initConfig(config *Config, ethrpcClient *ethrpc.Client) error {
	var (
		mainRegistryAddress, metaRegistryAddress, metaFactoryAddress, cryptoRegistryAddress, cryptoFactoryAddress common.Address
	)
	calls := ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []any{big.NewInt(0)},
	}, []any{&mainRegistryAddress}).AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		// id 7 = MetaRegistry per Curve's AddressProvider (get_id_info(7).description
		// == "Metaregistry"). Returns the zero address on chains without one, in which
		// case base-pool resolution falls back to the pool's own base_pool() getter.
		Params: []any{big.NewInt(7)},
	}, []any{&metaRegistryAddress}).AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []any{big.NewInt(3)},
	}, []any{&metaFactoryAddress}).AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []any{big.NewInt(5)},
	}, []any{&cryptoRegistryAddress}).AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: config.AddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []any{big.NewInt(6)},
	}, []any{&cryptoFactoryAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"addressProvider": config.AddressProvider,
			"error":           err,
		}).Errorf("failed to get address from address provider")
		return err
	}

	config.MainRegistryAddress = mainRegistryAddress.Hex()
	config.MetaRegistryAddress = metaRegistryAddress.Hex()
	config.MetaPoolsFactoryAddress = metaFactoryAddress.Hex()
	config.CryptoPoolsRegistryAddress = cryptoRegistryAddress.Hex()
	config.CryptoPoolsFactoryAddress = cryptoFactoryAddress.Hex()

	return nil
}

func getAPrecisions(aList, aPreciseList []*uint256.Int) ([]*big.Int, error) {
	var aPrecisions = make([]*big.Int, len(aList))
	for i := range aPrecisions {
		a := u256ToBig(aList[i])
		aPrecise := u256ToBig(aPreciseList[i])

		if a != nil && aPrecise != nil {
			aPrecisions[i] = new(big.Int).Div(aPrecise, a)
		} else if a != nil {
			aPrecisions[i] = big.NewInt(1)
		} else {
			return nil, errors.New("missing A data")
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
		s = append(s, hexutil.Encode(address[:]))
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
		return zeroString
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

func u256ToBig(num *uint256.Int) *big.Int {
	if num == nil {
		return nil
	}

	return num.ToBig()
}

func safeCastUint256ToString(num *uint256.Int) string {
	if num == nil {
		return zeroString
	}

	return num.String()
}

func safeCastUint256ToInt64(num *uint256.Int) int64 {
	if num == nil {
		return zero
	}

	return num.ToBig().Int64()
}

func safeCastUint256ToReserve(num *uint256.Int) string {
	if num == nil {
		return zeroString
	}

	return num.String()
}
