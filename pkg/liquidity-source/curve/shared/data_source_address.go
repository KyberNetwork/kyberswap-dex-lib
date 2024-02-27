package shared

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

var (
	DataSourceAddresses = map[string]map[CurveDataSource]string{}
)

func InitDataSourceAddresses(lg logger.Logger, config *Config, ethrpcClient *ethrpc.Client) error {
	if _, ok := DataSourceAddresses[config.ChainCode]; ok {
		return nil
	}
	lg.Info("fetching datasource addresses")

	DataSourceAddresses[config.ChainCode] = map[CurveDataSource]string{}

	// only get main registry address for now (to check custom rates)
	var mainRegistryAddress common.Address

	calls := ethrpcClient.NewRequest()

	calls.AddCall(&ethrpc.Call{
		ABI:    addressProviderABI,
		Target: CurveAddressProvider,
		Method: addressProviderMethodGetAddress,
		Params: []interface{}{big.NewInt(0)},
	}, []interface{}{&mainRegistryAddress})

	if _, err := calls.Aggregate(); err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get address from address provider")
		return err
	}

	DataSourceAddresses[config.ChainCode][CURVE_DATASOURCE_MAIN] = mainRegistryAddress.Hex()
	lg.Infof("fetched datasource addresses %v", DataSourceAddresses[config.ChainCode])
	return nil
}
