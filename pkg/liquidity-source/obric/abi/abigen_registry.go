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

// RegistryPoolRecord is an auto generated low-level Go binding around an user-defined struct.
type RegistryPoolRecord struct {
	XToken common.Address
	YToken common.Address
	Pool   common.Address
}

// ObricRegistryMetaData contains all meta data concerning the ObricRegistry contract.
var ObricRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"getPools\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"xToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"yToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"internalType\":\"structRegistry.PoolRecord[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"xToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"yToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"id\",\"type\":\"uint32\"}],\"name\":\"NewPoolEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"preKs\",\"type\":\"uint256[]\"}],\"name\":\"PoolsPreKEvent\",\"type\":\"event\"}]",
}

// ObricRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use ObricRegistryMetaData.ABI instead.
var ObricRegistryABI = ObricRegistryMetaData.ABI

// ObricRegistry is an auto generated Go binding around an Ethereum contract.
type ObricRegistry struct {
	ObricRegistryCaller     // Read-only binding to the contract
	ObricRegistryTransactor // Write-only binding to the contract
	ObricRegistryFilterer   // Log filterer for contract events
}

// ObricRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type ObricRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ObricRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ObricRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ObricRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ObricRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ObricRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ObricRegistrySession struct {
	Contract     *ObricRegistry    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ObricRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ObricRegistryCallerSession struct {
	Contract *ObricRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ObricRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ObricRegistryTransactorSession struct {
	Contract     *ObricRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ObricRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type ObricRegistryRaw struct {
	Contract *ObricRegistry // Generic contract binding to access the raw methods on
}

// ObricRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ObricRegistryCallerRaw struct {
	Contract *ObricRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// ObricRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ObricRegistryTransactorRaw struct {
	Contract *ObricRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewObricRegistry creates a new instance of ObricRegistry, bound to a specific deployed contract.
func NewObricRegistry(address common.Address, backend bind.ContractBackend) (*ObricRegistry, error) {
	contract, err := bindObricRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ObricRegistry{ObricRegistryCaller: ObricRegistryCaller{contract: contract}, ObricRegistryTransactor: ObricRegistryTransactor{contract: contract}, ObricRegistryFilterer: ObricRegistryFilterer{contract: contract}}, nil
}

// NewObricRegistryCaller creates a new read-only instance of ObricRegistry, bound to a specific deployed contract.
func NewObricRegistryCaller(address common.Address, caller bind.ContractCaller) (*ObricRegistryCaller, error) {
	contract, err := bindObricRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ObricRegistryCaller{contract: contract}, nil
}

// NewObricRegistryTransactor creates a new write-only instance of ObricRegistry, bound to a specific deployed contract.
func NewObricRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*ObricRegistryTransactor, error) {
	contract, err := bindObricRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ObricRegistryTransactor{contract: contract}, nil
}

// NewObricRegistryFilterer creates a new log filterer instance of ObricRegistry, bound to a specific deployed contract.
func NewObricRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*ObricRegistryFilterer, error) {
	contract, err := bindObricRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ObricRegistryFilterer{contract: contract}, nil
}

// bindObricRegistry binds a generic wrapper to an already deployed contract.
func bindObricRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ObricRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ObricRegistry *ObricRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ObricRegistry.Contract.ObricRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ObricRegistry *ObricRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ObricRegistry.Contract.ObricRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ObricRegistry *ObricRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ObricRegistry.Contract.ObricRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ObricRegistry *ObricRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ObricRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ObricRegistry *ObricRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ObricRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ObricRegistry *ObricRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ObricRegistry.Contract.contract.Transact(opts, method, params...)
}

// GetPools is a free data retrieval call binding the contract method 0x673a2a1f.
//
// Solidity: function getPools() view returns((address,address,address)[])
func (_ObricRegistry *ObricRegistryCaller) GetPools(opts *bind.CallOpts) ([]RegistryPoolRecord, error) {
	var out []interface{}
	err := _ObricRegistry.contract.Call(opts, &out, "getPools")

	if err != nil {
		return *new([]RegistryPoolRecord), err
	}

	out0 := *abi.ConvertType(out[0], new([]RegistryPoolRecord)).(*[]RegistryPoolRecord)

	return out0, err

}

// GetPools is a free data retrieval call binding the contract method 0x673a2a1f.
//
// Solidity: function getPools() view returns((address,address,address)[])
func (_ObricRegistry *ObricRegistrySession) GetPools() ([]RegistryPoolRecord, error) {
	return _ObricRegistry.Contract.GetPools(&_ObricRegistry.CallOpts)
}

// GetPools is a free data retrieval call binding the contract method 0x673a2a1f.
//
// Solidity: function getPools() view returns((address,address,address)[])
func (_ObricRegistry *ObricRegistryCallerSession) GetPools() ([]RegistryPoolRecord, error) {
	return _ObricRegistry.Contract.GetPools(&_ObricRegistry.CallOpts)
}

// ObricRegistryNewPoolEventIterator is returned from FilterNewPoolEvent and is used to iterate over the raw logs and unpacked data for NewPoolEvent events raised by the ObricRegistry contract.
type ObricRegistryNewPoolEventIterator struct {
	Event *ObricRegistryNewPoolEvent // Event containing the contract specifics and raw log

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
func (it *ObricRegistryNewPoolEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ObricRegistryNewPoolEvent)
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
		it.Event = new(ObricRegistryNewPoolEvent)
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
func (it *ObricRegistryNewPoolEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ObricRegistryNewPoolEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ObricRegistryNewPoolEvent represents a NewPoolEvent event raised by the ObricRegistry contract.
type ObricRegistryNewPoolEvent struct {
	XToken common.Address
	YToken common.Address
	Pool   common.Address
	Id     uint32
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterNewPoolEvent is a free log retrieval operation binding the contract event 0xcaac5c50f5397d5cad2a72728fa4bcdfe8c770bc19dc6890409e53d60b96857d.
//
// Solidity: event NewPoolEvent(address xToken, address yToken, address pool, uint32 id)
func (_ObricRegistry *ObricRegistryFilterer) FilterNewPoolEvent(opts *bind.FilterOpts) (*ObricRegistryNewPoolEventIterator, error) {

	logs, sub, err := _ObricRegistry.contract.FilterLogs(opts, "NewPoolEvent")
	if err != nil {
		return nil, err
	}
	return &ObricRegistryNewPoolEventIterator{contract: _ObricRegistry.contract, event: "NewPoolEvent", logs: logs, sub: sub}, nil
}

// WatchNewPoolEvent is a free log subscription operation binding the contract event 0xcaac5c50f5397d5cad2a72728fa4bcdfe8c770bc19dc6890409e53d60b96857d.
//
// Solidity: event NewPoolEvent(address xToken, address yToken, address pool, uint32 id)
func (_ObricRegistry *ObricRegistryFilterer) WatchNewPoolEvent(opts *bind.WatchOpts, sink chan<- *ObricRegistryNewPoolEvent) (event.Subscription, error) {

	logs, sub, err := _ObricRegistry.contract.WatchLogs(opts, "NewPoolEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ObricRegistryNewPoolEvent)
				if err := _ObricRegistry.contract.UnpackLog(event, "NewPoolEvent", log); err != nil {
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

// ParseNewPoolEvent is a log parse operation binding the contract event 0xcaac5c50f5397d5cad2a72728fa4bcdfe8c770bc19dc6890409e53d60b96857d.
//
// Solidity: event NewPoolEvent(address xToken, address yToken, address pool, uint32 id)
func (_ObricRegistry *ObricRegistryFilterer) ParseNewPoolEvent(log types.Log) (*ObricRegistryNewPoolEvent, error) {
	event := new(ObricRegistryNewPoolEvent)
	if err := _ObricRegistry.contract.UnpackLog(event, "NewPoolEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ObricRegistryPoolsPreKEventIterator is returned from FilterPoolsPreKEvent and is used to iterate over the raw logs and unpacked data for PoolsPreKEvent events raised by the ObricRegistry contract.
type ObricRegistryPoolsPreKEventIterator struct {
	Event *ObricRegistryPoolsPreKEvent // Event containing the contract specifics and raw log

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
func (it *ObricRegistryPoolsPreKEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ObricRegistryPoolsPreKEvent)
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
		it.Event = new(ObricRegistryPoolsPreKEvent)
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
func (it *ObricRegistryPoolsPreKEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ObricRegistryPoolsPreKEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ObricRegistryPoolsPreKEvent represents a PoolsPreKEvent event raised by the ObricRegistry contract.
type ObricRegistryPoolsPreKEvent struct {
	PreKs []*big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterPoolsPreKEvent is a free log retrieval operation binding the contract event 0x3166394b161ec8a4df959a00f9ba094ecd71070e9b9e9f3b28c3898ce9114629.
//
// Solidity: event PoolsPreKEvent(uint256[] preKs)
func (_ObricRegistry *ObricRegistryFilterer) FilterPoolsPreKEvent(opts *bind.FilterOpts) (*ObricRegistryPoolsPreKEventIterator, error) {

	logs, sub, err := _ObricRegistry.contract.FilterLogs(opts, "PoolsPreKEvent")
	if err != nil {
		return nil, err
	}
	return &ObricRegistryPoolsPreKEventIterator{contract: _ObricRegistry.contract, event: "PoolsPreKEvent", logs: logs, sub: sub}, nil
}

// WatchPoolsPreKEvent is a free log subscription operation binding the contract event 0x3166394b161ec8a4df959a00f9ba094ecd71070e9b9e9f3b28c3898ce9114629.
//
// Solidity: event PoolsPreKEvent(uint256[] preKs)
func (_ObricRegistry *ObricRegistryFilterer) WatchPoolsPreKEvent(opts *bind.WatchOpts, sink chan<- *ObricRegistryPoolsPreKEvent) (event.Subscription, error) {

	logs, sub, err := _ObricRegistry.contract.WatchLogs(opts, "PoolsPreKEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ObricRegistryPoolsPreKEvent)
				if err := _ObricRegistry.contract.UnpackLog(event, "PoolsPreKEvent", log); err != nil {
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

// ParsePoolsPreKEvent is a log parse operation binding the contract event 0x3166394b161ec8a4df959a00f9ba094ecd71070e9b9e9f3b28c3898ce9114629.
//
// Solidity: event PoolsPreKEvent(uint256[] preKs)
func (_ObricRegistry *ObricRegistryFilterer) ParsePoolsPreKEvent(log types.Log) (*ObricRegistryPoolsPreKEvent, error) {
	event := new(ObricRegistryPoolsPreKEvent)
	if err := _ObricRegistry.contract.UnpackLog(event, "PoolsPreKEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
