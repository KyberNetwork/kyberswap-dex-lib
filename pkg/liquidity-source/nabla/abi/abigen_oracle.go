// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abis

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// NablaOracleMetaData contains all meta data concerning the NablaOracle contract.
var NablaOracleMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"getAssetPrice\",\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getUpdateFee\",\"inputs\":[{\"name\":\"_updateData\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[{\"name\":\"updateFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isPriceFeedRegistered\",\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"isRegistered_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"AssetRegistered\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"assetName\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"priceFeedId\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BackstopPoolRemoved\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"backstopPool\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BackstopPoolSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"backstopPool\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}]},{\"type\":\"event\",\"name\":\"PriceFeedUpdate\",\"inputs\":[{\"name\":\"id\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"publishTime\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"price\",\"type\":\"int64\",\"indexed\":false,\"internalType\":\"int64\"},{\"name\":\"conf\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}]},{\"type\":\"event\",\"name\":\"PriceFeedsUpdated\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"updateFee\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"PriceMaxAgeSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nablaContract\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newPriceMaxAge\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SignerSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newSigner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"oldSigner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TokenRegistered\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"token\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"priceFeedId\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"assetName\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"}]}]",
}

// NablaOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use NablaOracleMetaData.ABI instead.
var NablaOracleABI = NablaOracleMetaData.ABI

// NablaOracle is an auto generated Go binding around an Ethereum contract.
type NablaOracle struct {
	NablaOracleCaller     // Read-only binding to the contract
	NablaOracleTransactor // Write-only binding to the contract
	NablaOracleFilterer   // Log filterer for contract events
}

// NablaOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type NablaOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NablaOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NablaOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NablaOracleSession struct {
	Contract     *NablaOracle      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NablaOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NablaOracleCallerSession struct {
	Contract *NablaOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// NablaOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NablaOracleTransactorSession struct {
	Contract     *NablaOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// NablaOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type NablaOracleRaw struct {
	Contract *NablaOracle // Generic contract binding to access the raw methods on
}

// NablaOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NablaOracleCallerRaw struct {
	Contract *NablaOracleCaller // Generic read-only contract binding to access the raw methods on
}

// NablaOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NablaOracleTransactorRaw struct {
	Contract *NablaOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNablaOracle creates a new instance of NablaOracle, bound to a specific deployed contract.
func NewNablaOracle(address common.Address, backend bind.ContractBackend) (*NablaOracle, error) {
	contract, err := bindNablaOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NablaOracle{NablaOracleCaller: NablaOracleCaller{contract: contract}, NablaOracleTransactor: NablaOracleTransactor{contract: contract}, NablaOracleFilterer: NablaOracleFilterer{contract: contract}}, nil
}

// NewNablaOracleCaller creates a new read-only instance of NablaOracle, bound to a specific deployed contract.
func NewNablaOracleCaller(address common.Address, caller bind.ContractCaller) (*NablaOracleCaller, error) {
	contract, err := bindNablaOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NablaOracleCaller{contract: contract}, nil
}

// NewNablaOracleTransactor creates a new write-only instance of NablaOracle, bound to a specific deployed contract.
func NewNablaOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*NablaOracleTransactor, error) {
	contract, err := bindNablaOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NablaOracleTransactor{contract: contract}, nil
}

// NewNablaOracleFilterer creates a new log filterer instance of NablaOracle, bound to a specific deployed contract.
func NewNablaOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*NablaOracleFilterer, error) {
	contract, err := bindNablaOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NablaOracleFilterer{contract: contract}, nil
}

// bindNablaOracle binds a generic wrapper to an already deployed contract.
func bindNablaOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NablaOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NablaOracle *NablaOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NablaOracle.Contract.NablaOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NablaOracle *NablaOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaOracle.Contract.NablaOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NablaOracle *NablaOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NablaOracle.Contract.NablaOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NablaOracle *NablaOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NablaOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NablaOracle *NablaOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NablaOracle *NablaOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NablaOracle.Contract.contract.Transact(opts, method, params...)
}

// GetAssetPrice is a free data retrieval call binding the contract method 0xb3596f07.
//
// Solidity: function getAssetPrice(address _asset) view returns(uint256)
func (_NablaOracle *NablaOracleCaller) GetAssetPrice(opts *bind.CallOpts, _asset common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NablaOracle.contract.Call(opts, &out, "getAssetPrice", _asset)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAssetPrice is a free data retrieval call binding the contract method 0xb3596f07.
//
// Solidity: function getAssetPrice(address _asset) view returns(uint256)
func (_NablaOracle *NablaOracleSession) GetAssetPrice(_asset common.Address) (*big.Int, error) {
	return _NablaOracle.Contract.GetAssetPrice(&_NablaOracle.CallOpts, _asset)
}

// GetAssetPrice is a free data retrieval call binding the contract method 0xb3596f07.
//
// Solidity: function getAssetPrice(address _asset) view returns(uint256)
func (_NablaOracle *NablaOracleCallerSession) GetAssetPrice(_asset common.Address) (*big.Int, error) {
	return _NablaOracle.Contract.GetAssetPrice(&_NablaOracle.CallOpts, _asset)
}

// GetUpdateFee is a free data retrieval call binding the contract method 0xd47eed45.
//
// Solidity: function getUpdateFee(bytes[] _updateData) view returns(uint256 updateFee_)
func (_NablaOracle *NablaOracleCaller) GetUpdateFee(opts *bind.CallOpts, _updateData [][]byte) (*big.Int, error) {
	var out []interface{}
	err := _NablaOracle.contract.Call(opts, &out, "getUpdateFee", _updateData)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetUpdateFee is a free data retrieval call binding the contract method 0xd47eed45.
//
// Solidity: function getUpdateFee(bytes[] _updateData) view returns(uint256 updateFee_)
func (_NablaOracle *NablaOracleSession) GetUpdateFee(_updateData [][]byte) (*big.Int, error) {
	return _NablaOracle.Contract.GetUpdateFee(&_NablaOracle.CallOpts, _updateData)
}

// GetUpdateFee is a free data retrieval call binding the contract method 0xd47eed45.
//
// Solidity: function getUpdateFee(bytes[] _updateData) view returns(uint256 updateFee_)
func (_NablaOracle *NablaOracleCallerSession) GetUpdateFee(_updateData [][]byte) (*big.Int, error) {
	return _NablaOracle.Contract.GetUpdateFee(&_NablaOracle.CallOpts, _updateData)
}

// IsPriceFeedRegistered is a free data retrieval call binding the contract method 0xd908f13f.
//
// Solidity: function isPriceFeedRegistered(address _asset) view returns(bool isRegistered_)
func (_NablaOracle *NablaOracleCaller) IsPriceFeedRegistered(opts *bind.CallOpts, _asset common.Address) (bool, error) {
	var out []interface{}
	err := _NablaOracle.contract.Call(opts, &out, "isPriceFeedRegistered", _asset)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPriceFeedRegistered is a free data retrieval call binding the contract method 0xd908f13f.
//
// Solidity: function isPriceFeedRegistered(address _asset) view returns(bool isRegistered_)
func (_NablaOracle *NablaOracleSession) IsPriceFeedRegistered(_asset common.Address) (bool, error) {
	return _NablaOracle.Contract.IsPriceFeedRegistered(&_NablaOracle.CallOpts, _asset)
}

// IsPriceFeedRegistered is a free data retrieval call binding the contract method 0xd908f13f.
//
// Solidity: function isPriceFeedRegistered(address _asset) view returns(bool isRegistered_)
func (_NablaOracle *NablaOracleCallerSession) IsPriceFeedRegistered(_asset common.Address) (bool, error) {
	return _NablaOracle.Contract.IsPriceFeedRegistered(&_NablaOracle.CallOpts, _asset)
}

// NablaOracleAssetRegisteredIterator is returned from FilterAssetRegistered and is used to iterate over the raw logs and unpacked data for AssetRegistered events raised by the NablaOracle contract.
type NablaOracleAssetRegisteredIterator struct {
	Event *NablaOracleAssetRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOracleAssetRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOracleAssetRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOracleAssetRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOracleAssetRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOracleAssetRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOracleAssetRegistered represents a AssetRegistered event raised by the NablaOracle contract.
type NablaOracleAssetRegistered struct {
	Sender      common.Address
	AssetName   string
	PriceFeedId [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetRegistered is a free log retrieval operation binding the contract event 0xeba6a64feb0f13ab3d0b94b34f2ddf30802ba0f4758ae5087f7080395e102967.
//
// Solidity: event AssetRegistered(address indexed sender, string assetName, bytes32 priceFeedId)
func (_NablaOracle *NablaOracleFilterer) FilterAssetRegistered(opts *bind.FilterOpts, sender []common.Address) (*NablaOracleAssetRegisteredIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "AssetRegistered", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaOracleAssetRegisteredIterator{contract: _NablaOracle.contract, event: "AssetRegistered", logs: logs, sub: sub}, nil
}

// WatchAssetRegistered is a free log subscription operation binding the contract event 0xeba6a64feb0f13ab3d0b94b34f2ddf30802ba0f4758ae5087f7080395e102967.
//
// Solidity: event AssetRegistered(address indexed sender, string assetName, bytes32 priceFeedId)
func (_NablaOracle *NablaOracleFilterer) WatchAssetRegistered(opts *bind.WatchOpts, sink chan<- *NablaOracleAssetRegistered, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "AssetRegistered", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOracleAssetRegistered)
				if err := _NablaOracle.contract.UnpackLog(event, "AssetRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAssetRegistered is a log parse operation binding the contract event 0xeba6a64feb0f13ab3d0b94b34f2ddf30802ba0f4758ae5087f7080395e102967.
//
// Solidity: event AssetRegistered(address indexed sender, string assetName, bytes32 priceFeedId)
func (_NablaOracle *NablaOracleFilterer) ParseAssetRegistered(log types.Log) (*NablaOracleAssetRegistered, error) {
	event := new(NablaOracleAssetRegistered)
	if err := _NablaOracle.contract.UnpackLog(event, "AssetRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOracleBackstopPoolRemovedIterator is returned from FilterBackstopPoolRemoved and is used to iterate over the raw logs and unpacked data for BackstopPoolRemoved events raised by the NablaOracle contract.
type NablaOracleBackstopPoolRemovedIterator struct {
	Event *NablaOracleBackstopPoolRemoved // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOracleBackstopPoolRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOracleBackstopPoolRemoved)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOracleBackstopPoolRemoved)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOracleBackstopPoolRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOracleBackstopPoolRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOracleBackstopPoolRemoved represents a BackstopPoolRemoved event raised by the NablaOracle contract.
type NablaOracleBackstopPoolRemoved struct {
	Sender       common.Address
	BackstopPool common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBackstopPoolRemoved is a free log retrieval operation binding the contract event 0xc54b6f390c37c58f1f0aa367f2f9bdb5e23c15fac79c3692b16f01c2e75d4888.
//
// Solidity: event BackstopPoolRemoved(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) FilterBackstopPoolRemoved(opts *bind.FilterOpts, sender []common.Address) (*NablaOracleBackstopPoolRemovedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "BackstopPoolRemoved", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaOracleBackstopPoolRemovedIterator{contract: _NablaOracle.contract, event: "BackstopPoolRemoved", logs: logs, sub: sub}, nil
}

// WatchBackstopPoolRemoved is a free log subscription operation binding the contract event 0xc54b6f390c37c58f1f0aa367f2f9bdb5e23c15fac79c3692b16f01c2e75d4888.
//
// Solidity: event BackstopPoolRemoved(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) WatchBackstopPoolRemoved(opts *bind.WatchOpts, sink chan<- *NablaOracleBackstopPoolRemoved, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "BackstopPoolRemoved", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOracleBackstopPoolRemoved)
				if err := _NablaOracle.contract.UnpackLog(event, "BackstopPoolRemoved", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBackstopPoolRemoved is a log parse operation binding the contract event 0xc54b6f390c37c58f1f0aa367f2f9bdb5e23c15fac79c3692b16f01c2e75d4888.
//
// Solidity: event BackstopPoolRemoved(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) ParseBackstopPoolRemoved(log types.Log) (*NablaOracleBackstopPoolRemoved, error) {
	event := new(NablaOracleBackstopPoolRemoved)
	if err := _NablaOracle.contract.UnpackLog(event, "BackstopPoolRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOracleBackstopPoolSetIterator is returned from FilterBackstopPoolSet and is used to iterate over the raw logs and unpacked data for BackstopPoolSet events raised by the NablaOracle contract.
type NablaOracleBackstopPoolSetIterator struct {
	Event *NablaOracleBackstopPoolSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOracleBackstopPoolSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOracleBackstopPoolSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOracleBackstopPoolSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOracleBackstopPoolSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOracleBackstopPoolSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOracleBackstopPoolSet represents a BackstopPoolSet event raised by the NablaOracle contract.
type NablaOracleBackstopPoolSet struct {
	Sender       common.Address
	BackstopPool common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBackstopPoolSet is a free log retrieval operation binding the contract event 0x693ba3abe2973ba4de5b3a7453ce8c5e47b4ed89b6a367c7d2efaea5244ad2ff.
//
// Solidity: event BackstopPoolSet(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) FilterBackstopPoolSet(opts *bind.FilterOpts, sender []common.Address) (*NablaOracleBackstopPoolSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "BackstopPoolSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaOracleBackstopPoolSetIterator{contract: _NablaOracle.contract, event: "BackstopPoolSet", logs: logs, sub: sub}, nil
}

// WatchBackstopPoolSet is a free log subscription operation binding the contract event 0x693ba3abe2973ba4de5b3a7453ce8c5e47b4ed89b6a367c7d2efaea5244ad2ff.
//
// Solidity: event BackstopPoolSet(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) WatchBackstopPoolSet(opts *bind.WatchOpts, sink chan<- *NablaOracleBackstopPoolSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "BackstopPoolSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOracleBackstopPoolSet)
				if err := _NablaOracle.contract.UnpackLog(event, "BackstopPoolSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBackstopPoolSet is a log parse operation binding the contract event 0x693ba3abe2973ba4de5b3a7453ce8c5e47b4ed89b6a367c7d2efaea5244ad2ff.
//
// Solidity: event BackstopPoolSet(address indexed sender, address backstopPool)
func (_NablaOracle *NablaOracleFilterer) ParseBackstopPoolSet(log types.Log) (*NablaOracleBackstopPoolSet, error) {
	event := new(NablaOracleBackstopPoolSet)
	if err := _NablaOracle.contract.UnpackLog(event, "BackstopPoolSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOraclePriceFeedUpdateIterator is returned from FilterPriceFeedUpdate and is used to iterate over the raw logs and unpacked data for PriceFeedUpdate events raised by the NablaOracle contract.
type NablaOraclePriceFeedUpdateIterator struct {
	Event *NablaOraclePriceFeedUpdate // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOraclePriceFeedUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOraclePriceFeedUpdate)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOraclePriceFeedUpdate)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOraclePriceFeedUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOraclePriceFeedUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOraclePriceFeedUpdate represents a PriceFeedUpdate event raised by the NablaOracle contract.
type NablaOraclePriceFeedUpdate struct {
	Id          [32]byte
	PublishTime uint64
	Price       int64
	Conf        uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPriceFeedUpdate is a free log retrieval operation binding the contract event 0xd06a6b7f4918494b3719217d1802786c1f5112a6c1d88fe2cfec00b4584f6aec.
//
// Solidity: event PriceFeedUpdate(bytes32 indexed id, uint64 publishTime, int64 price, uint64 conf)
func (_NablaOracle *NablaOracleFilterer) FilterPriceFeedUpdate(opts *bind.FilterOpts, id [][32]byte) (*NablaOraclePriceFeedUpdateIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "PriceFeedUpdate", idRule)
	if err != nil {
		return nil, err
	}
	return &NablaOraclePriceFeedUpdateIterator{contract: _NablaOracle.contract, event: "PriceFeedUpdate", logs: logs, sub: sub}, nil
}

// WatchPriceFeedUpdate is a free log subscription operation binding the contract event 0xd06a6b7f4918494b3719217d1802786c1f5112a6c1d88fe2cfec00b4584f6aec.
//
// Solidity: event PriceFeedUpdate(bytes32 indexed id, uint64 publishTime, int64 price, uint64 conf)
func (_NablaOracle *NablaOracleFilterer) WatchPriceFeedUpdate(opts *bind.WatchOpts, sink chan<- *NablaOraclePriceFeedUpdate, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "PriceFeedUpdate", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOraclePriceFeedUpdate)
				if err := _NablaOracle.contract.UnpackLog(event, "PriceFeedUpdate", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePriceFeedUpdate is a log parse operation binding the contract event 0xd06a6b7f4918494b3719217d1802786c1f5112a6c1d88fe2cfec00b4584f6aec.
//
// Solidity: event PriceFeedUpdate(bytes32 indexed id, uint64 publishTime, int64 price, uint64 conf)
func (_NablaOracle *NablaOracleFilterer) ParsePriceFeedUpdate(log types.Log) (*NablaOraclePriceFeedUpdate, error) {
	event := new(NablaOraclePriceFeedUpdate)
	if err := _NablaOracle.contract.UnpackLog(event, "PriceFeedUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOraclePriceFeedsUpdatedIterator is returned from FilterPriceFeedsUpdated and is used to iterate over the raw logs and unpacked data for PriceFeedsUpdated events raised by the NablaOracle contract.
type NablaOraclePriceFeedsUpdatedIterator struct {
	Event *NablaOraclePriceFeedsUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOraclePriceFeedsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOraclePriceFeedsUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOraclePriceFeedsUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOraclePriceFeedsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOraclePriceFeedsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOraclePriceFeedsUpdated represents a PriceFeedsUpdated event raised by the NablaOracle contract.
type NablaOraclePriceFeedsUpdated struct {
	Sender    common.Address
	UpdateFee *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPriceFeedsUpdated is a free log retrieval operation binding the contract event 0x6a03c66c27da2fc6f6d60da65cdc5f9328794ff707bbd89636c453fa407ac841.
//
// Solidity: event PriceFeedsUpdated(address indexed sender, uint256 updateFee)
func (_NablaOracle *NablaOracleFilterer) FilterPriceFeedsUpdated(opts *bind.FilterOpts, sender []common.Address) (*NablaOraclePriceFeedsUpdatedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "PriceFeedsUpdated", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaOraclePriceFeedsUpdatedIterator{contract: _NablaOracle.contract, event: "PriceFeedsUpdated", logs: logs, sub: sub}, nil
}

// WatchPriceFeedsUpdated is a free log subscription operation binding the contract event 0x6a03c66c27da2fc6f6d60da65cdc5f9328794ff707bbd89636c453fa407ac841.
//
// Solidity: event PriceFeedsUpdated(address indexed sender, uint256 updateFee)
func (_NablaOracle *NablaOracleFilterer) WatchPriceFeedsUpdated(opts *bind.WatchOpts, sink chan<- *NablaOraclePriceFeedsUpdated, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "PriceFeedsUpdated", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOraclePriceFeedsUpdated)
				if err := _NablaOracle.contract.UnpackLog(event, "PriceFeedsUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePriceFeedsUpdated is a log parse operation binding the contract event 0x6a03c66c27da2fc6f6d60da65cdc5f9328794ff707bbd89636c453fa407ac841.
//
// Solidity: event PriceFeedsUpdated(address indexed sender, uint256 updateFee)
func (_NablaOracle *NablaOracleFilterer) ParsePriceFeedsUpdated(log types.Log) (*NablaOraclePriceFeedsUpdated, error) {
	event := new(NablaOraclePriceFeedsUpdated)
	if err := _NablaOracle.contract.UnpackLog(event, "PriceFeedsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOraclePriceMaxAgeSetIterator is returned from FilterPriceMaxAgeSet and is used to iterate over the raw logs and unpacked data for PriceMaxAgeSet events raised by the NablaOracle contract.
type NablaOraclePriceMaxAgeSetIterator struct {
	Event *NablaOraclePriceMaxAgeSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOraclePriceMaxAgeSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOraclePriceMaxAgeSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOraclePriceMaxAgeSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOraclePriceMaxAgeSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOraclePriceMaxAgeSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOraclePriceMaxAgeSet represents a PriceMaxAgeSet event raised by the NablaOracle contract.
type NablaOraclePriceMaxAgeSet struct {
	Sender         common.Address
	NablaContract  common.Address
	NewPriceMaxAge *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterPriceMaxAgeSet is a free log retrieval operation binding the contract event 0x27c94e789a85b6ff917113c353f02b7ea440f3751d5d172a8a8afb1f6bbe5703.
//
// Solidity: event PriceMaxAgeSet(address indexed sender, address indexed nablaContract, uint256 newPriceMaxAge)
func (_NablaOracle *NablaOracleFilterer) FilterPriceMaxAgeSet(opts *bind.FilterOpts, sender []common.Address, nablaContract []common.Address) (*NablaOraclePriceMaxAgeSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var nablaContractRule []interface{}
	for _, nablaContractItem := range nablaContract {
		nablaContractRule = append(nablaContractRule, nablaContractItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "PriceMaxAgeSet", senderRule, nablaContractRule)
	if err != nil {
		return nil, err
	}
	return &NablaOraclePriceMaxAgeSetIterator{contract: _NablaOracle.contract, event: "PriceMaxAgeSet", logs: logs, sub: sub}, nil
}

// WatchPriceMaxAgeSet is a free log subscription operation binding the contract event 0x27c94e789a85b6ff917113c353f02b7ea440f3751d5d172a8a8afb1f6bbe5703.
//
// Solidity: event PriceMaxAgeSet(address indexed sender, address indexed nablaContract, uint256 newPriceMaxAge)
func (_NablaOracle *NablaOracleFilterer) WatchPriceMaxAgeSet(opts *bind.WatchOpts, sink chan<- *NablaOraclePriceMaxAgeSet, sender []common.Address, nablaContract []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var nablaContractRule []interface{}
	for _, nablaContractItem := range nablaContract {
		nablaContractRule = append(nablaContractRule, nablaContractItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "PriceMaxAgeSet", senderRule, nablaContractRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOraclePriceMaxAgeSet)
				if err := _NablaOracle.contract.UnpackLog(event, "PriceMaxAgeSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePriceMaxAgeSet is a log parse operation binding the contract event 0x27c94e789a85b6ff917113c353f02b7ea440f3751d5d172a8a8afb1f6bbe5703.
//
// Solidity: event PriceMaxAgeSet(address indexed sender, address indexed nablaContract, uint256 newPriceMaxAge)
func (_NablaOracle *NablaOracleFilterer) ParsePriceMaxAgeSet(log types.Log) (*NablaOraclePriceMaxAgeSet, error) {
	event := new(NablaOraclePriceMaxAgeSet)
	if err := _NablaOracle.contract.UnpackLog(event, "PriceMaxAgeSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOracleSignerSetIterator is returned from FilterSignerSet and is used to iterate over the raw logs and unpacked data for SignerSet events raised by the NablaOracle contract.
type NablaOracleSignerSetIterator struct {
	Event *NablaOracleSignerSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOracleSignerSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOracleSignerSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOracleSignerSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOracleSignerSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOracleSignerSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOracleSignerSet represents a SignerSet event raised by the NablaOracle contract.
type NablaOracleSignerSet struct {
	Sender    common.Address
	NewSigner common.Address
	OldSigner common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSignerSet is a free log retrieval operation binding the contract event 0x01e7d9da6e5a480e65545d59959ae63a2ae2251f641f46f4039f73ad9e1bb700.
//
// Solidity: event SignerSet(address indexed sender, address indexed newSigner, address indexed oldSigner)
func (_NablaOracle *NablaOracleFilterer) FilterSignerSet(opts *bind.FilterOpts, sender []common.Address, newSigner []common.Address, oldSigner []common.Address) (*NablaOracleSignerSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var newSignerRule []interface{}
	for _, newSignerItem := range newSigner {
		newSignerRule = append(newSignerRule, newSignerItem)
	}
	var oldSignerRule []interface{}
	for _, oldSignerItem := range oldSigner {
		oldSignerRule = append(oldSignerRule, oldSignerItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "SignerSet", senderRule, newSignerRule, oldSignerRule)
	if err != nil {
		return nil, err
	}
	return &NablaOracleSignerSetIterator{contract: _NablaOracle.contract, event: "SignerSet", logs: logs, sub: sub}, nil
}

// WatchSignerSet is a free log subscription operation binding the contract event 0x01e7d9da6e5a480e65545d59959ae63a2ae2251f641f46f4039f73ad9e1bb700.
//
// Solidity: event SignerSet(address indexed sender, address indexed newSigner, address indexed oldSigner)
func (_NablaOracle *NablaOracleFilterer) WatchSignerSet(opts *bind.WatchOpts, sink chan<- *NablaOracleSignerSet, sender []common.Address, newSigner []common.Address, oldSigner []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var newSignerRule []interface{}
	for _, newSignerItem := range newSigner {
		newSignerRule = append(newSignerRule, newSignerItem)
	}
	var oldSignerRule []interface{}
	for _, oldSignerItem := range oldSigner {
		oldSignerRule = append(oldSignerRule, oldSignerItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "SignerSet", senderRule, newSignerRule, oldSignerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOracleSignerSet)
				if err := _NablaOracle.contract.UnpackLog(event, "SignerSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSignerSet is a log parse operation binding the contract event 0x01e7d9da6e5a480e65545d59959ae63a2ae2251f641f46f4039f73ad9e1bb700.
//
// Solidity: event SignerSet(address indexed sender, address indexed newSigner, address indexed oldSigner)
func (_NablaOracle *NablaOracleFilterer) ParseSignerSet(log types.Log) (*NablaOracleSignerSet, error) {
	event := new(NablaOracleSignerSet)
	if err := _NablaOracle.contract.UnpackLog(event, "SignerSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaOracleTokenRegisteredIterator is returned from FilterTokenRegistered and is used to iterate over the raw logs and unpacked data for TokenRegistered events raised by the NablaOracle contract.
type NablaOracleTokenRegisteredIterator struct {
	Event *NablaOracleTokenRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NablaOracleTokenRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaOracleTokenRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NablaOracleTokenRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NablaOracleTokenRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaOracleTokenRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaOracleTokenRegistered represents a TokenRegistered event raised by the NablaOracle contract.
type NablaOracleTokenRegistered struct {
	Sender      common.Address
	Token       common.Address
	PriceFeedId [32]byte
	AssetName   string
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTokenRegistered is a free log retrieval operation binding the contract event 0x4ab9363f80f065eb4a2817b0ab892f125eac5b247b5ccb0b766cfb6381b3ecaa.
//
// Solidity: event TokenRegistered(address indexed sender, address token, bytes32 priceFeedId, string assetName)
func (_NablaOracle *NablaOracleFilterer) FilterTokenRegistered(opts *bind.FilterOpts, sender []common.Address) (*NablaOracleTokenRegisteredIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.FilterLogs(opts, "TokenRegistered", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaOracleTokenRegisteredIterator{contract: _NablaOracle.contract, event: "TokenRegistered", logs: logs, sub: sub}, nil
}

// WatchTokenRegistered is a free log subscription operation binding the contract event 0x4ab9363f80f065eb4a2817b0ab892f125eac5b247b5ccb0b766cfb6381b3ecaa.
//
// Solidity: event TokenRegistered(address indexed sender, address token, bytes32 priceFeedId, string assetName)
func (_NablaOracle *NablaOracleFilterer) WatchTokenRegistered(opts *bind.WatchOpts, sink chan<- *NablaOracleTokenRegistered, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaOracle.contract.WatchLogs(opts, "TokenRegistered", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaOracleTokenRegistered)
				if err := _NablaOracle.contract.UnpackLog(event, "TokenRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenRegistered is a log parse operation binding the contract event 0x4ab9363f80f065eb4a2817b0ab892f125eac5b247b5ccb0b766cfb6381b3ecaa.
//
// Solidity: event TokenRegistered(address indexed sender, address token, bytes32 priceFeedId, string assetName)
func (_NablaOracle *NablaOracleFilterer) ParseTokenRegistered(log types.Log) (*NablaOracleTokenRegistered, error) {
	event := new(NablaOracleTokenRegistered)
	if err := _NablaOracle.contract.UnpackLog(event, "TokenRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
