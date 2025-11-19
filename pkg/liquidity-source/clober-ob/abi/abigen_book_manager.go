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

// IBookManagerBookKey is an auto generated low-level Go binding around an user-defined struct.
type IBookManagerBookKey struct {
	Base        common.Address
	UnitSize    uint64
	Quote       common.Address
	MakerPolicy *big.Int
	Hooks       common.Address
	TakerPolicy *big.Int
}

// IBookManagerOrderInfo is an auto generated low-level Go binding around an user-defined struct.
type IBookManagerOrderInfo struct {
	Provider  common.Address
	Open      uint64
	Claimable uint64
}

// BookManagerMetaData contains all meta data concerning the BookManager contract.
var BookManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"BookNotOpened\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CurrencyNotSettled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ECDSAInvalidSignature\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"ECDSAInvalidSignatureLength\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"ECDSAInvalidSignatureS\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ERC20TransferFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"ERC721IncorrectOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ERC721InsufficientApproval\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"approver\",\"type\":\"address\"}],\"name\":\"ERC721InvalidApprover\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"ERC721InvalidOperator\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"ERC721InvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"}],\"name\":\"ERC721InvalidReceiver\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"ERC721InvalidSender\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ERC721NonexistentToken\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FailedHookCall\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hooks\",\"type\":\"address\"}],\"name\":\"HookAddressNotValid\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidFeePolicy\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHookResponse\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"InvalidProvider\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidShortString\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidTick\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidUnitSize\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"LockedBy\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NativeTransferFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PermitExpired\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"int256\",\"name\":\"value\",\"type\":\"int256\"}],\"name\":\"SafeCastOverflowedIntToUint\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"SafeCastOverflowedUintToInt\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"str\",\"type\":\"string\"}],\"name\":\"StringTooLong\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"approved\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"OrderId\",\"name\":\"orderId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"unit\",\"type\":\"uint64\"}],\"name\":\"Cancel\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"OrderId\",\"name\":\"orderId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"unit\",\"type\":\"uint64\"}],\"name\":\"Claim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Collect\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"Delist\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"EIP712DomainChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"BookId\",\"name\":\"bookId\",\"type\":\"uint192\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"Tick\",\"name\":\"tick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"orderIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"unit\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"Make\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"base\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"quote\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"unitSize\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"FeePolicy\",\"name\":\"makerPolicy\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"FeePolicy\",\"name\":\"takerPolicy\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"name\":\"Open\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferStarted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newDefaultProvider\",\"type\":\"address\"}],\"name\":\"SetDefaultProvider\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"BookId\",\"name\":\"bookId\",\"type\":\"uint192\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"Tick\",\"name\":\"tick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"unit\",\"type\":\"uint64\"}],\"name\":\"Take\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"Whitelist\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"baseURI\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"checkAuthorized\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"contractURI\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultProvider\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"base\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"unitSize\",\"type\":\"uint64\"},{\"internalType\":\"Currency\",\"name\":\"quote\",\"type\":\"address\"},{\"internalType\":\"FeePolicy\",\"name\":\"makerPolicy\",\"type\":\"uint24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"FeePolicy\",\"name\":\"takerPolicy\",\"type\":\"uint24\"}],\"internalType\":\"structIBookManager.BookKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"encodeBookKey\",\"outputs\":[{\"internalType\":\"BookId\",\"name\":\"\",\"type\":\"uint192\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"}],\"name\":\"getBookKey\",\"outputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"base\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"unitSize\",\"type\":\"uint64\"},{\"internalType\":\"Currency\",\"name\":\"quote\",\"type\":\"address\"},{\"internalType\":\"FeePolicy\",\"name\":\"makerPolicy\",\"type\":\"uint24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"FeePolicy\",\"name\":\"takerPolicy\",\"type\":\"uint24\"}],\"internalType\":\"structIBookManager.BookKey\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"getCurrencyDelta\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"},{\"internalType\":\"Tick\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"getDepth\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"}],\"name\":\"getHighest\",\"outputs\":[{\"internalType\":\"Tick\",\"type\":\"int32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"i\",\"type\":\"uint256\"}],\"name\":\"getLock\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLockData\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"OrderId\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"getOrder\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"open\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"claimable\",\"type\":\"uint64\"}],\"internalType\":\"structIBookManager.OrderInfo\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"}],\"name\":\"isEmpty\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"}],\"name\":\"isOpened\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"isWhitelisted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"load\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"startSlot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"nSlot\",\"type\":\"uint256\"}],\"name\":\"load\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"value\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"BookId\",\"name\":\"id\",\"type\":\"uint192\"},{\"internalType\":\"Tick\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"maxLessThan\",\"outputs\":[{\"internalType\":\"Tick\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"reservesOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"tokenOwed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"tokenURI\",\"outputs\":[{\"internalType\":\"string\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BookManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use BookManagerMetaData.ABI instead.
var BookManagerABI = BookManagerMetaData.ABI

// BookManager is an auto generated Go binding around an Ethereum contract.
type BookManager struct {
	BookManagerCaller     // Read-only binding to the contract
	BookManagerTransactor // Write-only binding to the contract
	BookManagerFilterer   // Log filterer for contract events
}

// BookManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type BookManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BookManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BookManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BookManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BookManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BookManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BookManagerSession struct {
	Contract     *BookManager      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BookManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BookManagerCallerSession struct {
	Contract *BookManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// BookManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BookManagerTransactorSession struct {
	Contract     *BookManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// BookManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type BookManagerRaw struct {
	Contract *BookManager // Generic contract binding to access the raw methods on
}

// BookManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BookManagerCallerRaw struct {
	Contract *BookManagerCaller // Generic read-only contract binding to access the raw methods on
}

// BookManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BookManagerTransactorRaw struct {
	Contract *BookManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBookManager creates a new instance of BookManager, bound to a specific deployed contract.
func NewBookManager(address common.Address, backend bind.ContractBackend) (*BookManager, error) {
	contract, err := bindBookManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BookManager{BookManagerCaller: BookManagerCaller{contract: contract}, BookManagerTransactor: BookManagerTransactor{contract: contract}, BookManagerFilterer: BookManagerFilterer{contract: contract}}, nil
}

// NewBookManagerCaller creates a new read-only instance of BookManager, bound to a specific deployed contract.
func NewBookManagerCaller(address common.Address, caller bind.ContractCaller) (*BookManagerCaller, error) {
	contract, err := bindBookManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BookManagerCaller{contract: contract}, nil
}

// NewBookManagerTransactor creates a new write-only instance of BookManager, bound to a specific deployed contract.
func NewBookManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*BookManagerTransactor, error) {
	contract, err := bindBookManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BookManagerTransactor{contract: contract}, nil
}

// NewBookManagerFilterer creates a new log filterer instance of BookManager, bound to a specific deployed contract.
func NewBookManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*BookManagerFilterer, error) {
	contract, err := bindBookManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BookManagerFilterer{contract: contract}, nil
}

// bindBookManager binds a generic wrapper to an already deployed contract.
func bindBookManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BookManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BookManager *BookManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BookManager.Contract.BookManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BookManager *BookManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BookManager.Contract.BookManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BookManager *BookManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BookManager.Contract.BookManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BookManager *BookManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BookManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BookManager *BookManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BookManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BookManager *BookManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BookManager.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_BookManager *BookManagerCaller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_BookManager *BookManagerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _BookManager.Contract.BalanceOf(&_BookManager.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_BookManager *BookManagerCallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _BookManager.Contract.BalanceOf(&_BookManager.CallOpts, owner)
}

// BaseURI is a free data retrieval call binding the contract method 0x6c0360eb.
//
// Solidity: function baseURI() view returns(string)
func (_BookManager *BookManagerCaller) BaseURI(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "baseURI")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// BaseURI is a free data retrieval call binding the contract method 0x6c0360eb.
//
// Solidity: function baseURI() view returns(string)
func (_BookManager *BookManagerSession) BaseURI() (string, error) {
	return _BookManager.Contract.BaseURI(&_BookManager.CallOpts)
}

// BaseURI is a free data retrieval call binding the contract method 0x6c0360eb.
//
// Solidity: function baseURI() view returns(string)
func (_BookManager *BookManagerCallerSession) BaseURI() (string, error) {
	return _BookManager.Contract.BaseURI(&_BookManager.CallOpts)
}

// CheckAuthorized is a free data retrieval call binding the contract method 0x2f584a6d.
//
// Solidity: function checkAuthorized(address owner, address spender, uint256 tokenId) view returns()
func (_BookManager *BookManagerCaller) CheckAuthorized(opts *bind.CallOpts, owner common.Address, spender common.Address, tokenId *big.Int) error {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "checkAuthorized", owner, spender, tokenId)

	if err != nil {
		return err
	}

	return err

}

// CheckAuthorized is a free data retrieval call binding the contract method 0x2f584a6d.
//
// Solidity: function checkAuthorized(address owner, address spender, uint256 tokenId) view returns()
func (_BookManager *BookManagerSession) CheckAuthorized(owner common.Address, spender common.Address, tokenId *big.Int) error {
	return _BookManager.Contract.CheckAuthorized(&_BookManager.CallOpts, owner, spender, tokenId)
}

// CheckAuthorized is a free data retrieval call binding the contract method 0x2f584a6d.
//
// Solidity: function checkAuthorized(address owner, address spender, uint256 tokenId) view returns()
func (_BookManager *BookManagerCallerSession) CheckAuthorized(owner common.Address, spender common.Address, tokenId *big.Int) error {
	return _BookManager.Contract.CheckAuthorized(&_BookManager.CallOpts, owner, spender, tokenId)
}

// ContractURI is a free data retrieval call binding the contract method 0xe8a3d485.
//
// Solidity: function contractURI() view returns(string)
func (_BookManager *BookManagerCaller) ContractURI(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "contractURI")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ContractURI is a free data retrieval call binding the contract method 0xe8a3d485.
//
// Solidity: function contractURI() view returns(string)
func (_BookManager *BookManagerSession) ContractURI() (string, error) {
	return _BookManager.Contract.ContractURI(&_BookManager.CallOpts)
}

// ContractURI is a free data retrieval call binding the contract method 0xe8a3d485.
//
// Solidity: function contractURI() view returns(string)
func (_BookManager *BookManagerCallerSession) ContractURI() (string, error) {
	return _BookManager.Contract.ContractURI(&_BookManager.CallOpts)
}

// DefaultProvider is a free data retrieval call binding the contract method 0xd83747e8.
//
// Solidity: function defaultProvider() view returns(address)
func (_BookManager *BookManagerCaller) DefaultProvider(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "defaultProvider")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultProvider is a free data retrieval call binding the contract method 0xd83747e8.
//
// Solidity: function defaultProvider() view returns(address)
func (_BookManager *BookManagerSession) DefaultProvider() (common.Address, error) {
	return _BookManager.Contract.DefaultProvider(&_BookManager.CallOpts)
}

// DefaultProvider is a free data retrieval call binding the contract method 0xd83747e8.
//
// Solidity: function defaultProvider() view returns(address)
func (_BookManager *BookManagerCallerSession) DefaultProvider() (common.Address, error) {
	return _BookManager.Contract.DefaultProvider(&_BookManager.CallOpts)
}

// EncodeBookKey is a free data retrieval call binding the contract method 0x1ff63f93.
//
// Solidity: function encodeBookKey((address,uint64,address,uint24,address,uint24) key) pure returns(uint192)
func (_BookManager *BookManagerCaller) EncodeBookKey(opts *bind.CallOpts, key IBookManagerBookKey) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "encodeBookKey", key)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EncodeBookKey is a free data retrieval call binding the contract method 0x1ff63f93.
//
// Solidity: function encodeBookKey((address,uint64,address,uint24,address,uint24) key) pure returns(uint192)
func (_BookManager *BookManagerSession) EncodeBookKey(key IBookManagerBookKey) (*big.Int, error) {
	return _BookManager.Contract.EncodeBookKey(&_BookManager.CallOpts, key)
}

// EncodeBookKey is a free data retrieval call binding the contract method 0x1ff63f93.
//
// Solidity: function encodeBookKey((address,uint64,address,uint24,address,uint24) key) pure returns(uint192)
func (_BookManager *BookManagerCallerSession) EncodeBookKey(key IBookManagerBookKey) (*big.Int, error) {
	return _BookManager.Contract.EncodeBookKey(&_BookManager.CallOpts, key)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _BookManager.Contract.GetApproved(&_BookManager.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _BookManager.Contract.GetApproved(&_BookManager.CallOpts, tokenId)
}

// GetBookKey is a free data retrieval call binding the contract method 0x9b22917d.
//
// Solidity: function getBookKey(uint192 id) view returns((address,uint64,address,uint24,address,uint24))
func (_BookManager *BookManagerCaller) GetBookKey(opts *bind.CallOpts, id *big.Int) (IBookManagerBookKey, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getBookKey", id)

	if err != nil {
		return *new(IBookManagerBookKey), err
	}

	out0 := *abi.ConvertType(out[0], new(IBookManagerBookKey)).(*IBookManagerBookKey)

	return out0, err

}

// GetBookKey is a free data retrieval call binding the contract method 0x9b22917d.
//
// Solidity: function getBookKey(uint192 id) view returns((address,uint64,address,uint24,address,uint24))
func (_BookManager *BookManagerSession) GetBookKey(id *big.Int) (IBookManagerBookKey, error) {
	return _BookManager.Contract.GetBookKey(&_BookManager.CallOpts, id)
}

// GetBookKey is a free data retrieval call binding the contract method 0x9b22917d.
//
// Solidity: function getBookKey(uint192 id) view returns((address,uint64,address,uint24,address,uint24))
func (_BookManager *BookManagerCallerSession) GetBookKey(id *big.Int) (IBookManagerBookKey, error) {
	return _BookManager.Contract.GetBookKey(&_BookManager.CallOpts, id)
}

// GetCurrencyDelta is a free data retrieval call binding the contract method 0x9611cf6c.
//
// Solidity: function getCurrencyDelta(address locker, address currency) view returns(int256)
func (_BookManager *BookManagerCaller) GetCurrencyDelta(opts *bind.CallOpts, locker common.Address, currency common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getCurrencyDelta", locker, currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrencyDelta is a free data retrieval call binding the contract method 0x9611cf6c.
//
// Solidity: function getCurrencyDelta(address locker, address currency) view returns(int256)
func (_BookManager *BookManagerSession) GetCurrencyDelta(locker common.Address, currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.GetCurrencyDelta(&_BookManager.CallOpts, locker, currency)
}

// GetCurrencyDelta is a free data retrieval call binding the contract method 0x9611cf6c.
//
// Solidity: function getCurrencyDelta(address locker, address currency) view returns(int256)
func (_BookManager *BookManagerCallerSession) GetCurrencyDelta(locker common.Address, currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.GetCurrencyDelta(&_BookManager.CallOpts, locker, currency)
}

// GetDepth is a free data retrieval call binding the contract method 0x41a8bb88.
//
// Solidity: function getDepth(uint192 id, int24 tick) view returns(uint64)
func (_BookManager *BookManagerCaller) GetDepth(opts *bind.CallOpts, id *big.Int, tick *big.Int) (uint64, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getDepth", id, tick)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetDepth is a free data retrieval call binding the contract method 0x41a8bb88.
//
// Solidity: function getDepth(uint192 id, int24 tick) view returns(uint64)
func (_BookManager *BookManagerSession) GetDepth(id *big.Int, tick *big.Int) (uint64, error) {
	return _BookManager.Contract.GetDepth(&_BookManager.CallOpts, id, tick)
}

// GetDepth is a free data retrieval call binding the contract method 0x41a8bb88.
//
// Solidity: function getDepth(uint192 id, int24 tick) view returns(uint64)
func (_BookManager *BookManagerCallerSession) GetDepth(id *big.Int, tick *big.Int) (uint64, error) {
	return _BookManager.Contract.GetDepth(&_BookManager.CallOpts, id, tick)
}

// GetHighest is a free data retrieval call binding the contract method 0xcdc92f2d.
//
// Solidity: function getHighest(uint192 id) view returns(int32)
func (_BookManager *BookManagerCaller) GetHighest(opts *bind.CallOpts, id *big.Int) (int32, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getHighest", id)

	if err != nil {
		return *new(int32), err
	}

	out0 := *abi.ConvertType(out[0], new(int32)).(*int32)

	return out0, err

}

// GetHighest is a free data retrieval call binding the contract method 0xcdc92f2d.
//
// Solidity: function getHighest(uint192 id) view returns(int32)
func (_BookManager *BookManagerSession) GetHighest(id *big.Int) (int32, error) {
	return _BookManager.Contract.GetHighest(&_BookManager.CallOpts, id)
}

// GetHighest is a free data retrieval call binding the contract method 0xcdc92f2d.
//
// Solidity: function getHighest(uint192 id) view returns(int32)
func (_BookManager *BookManagerCallerSession) GetHighest(id *big.Int) (int32, error) {
	return _BookManager.Contract.GetHighest(&_BookManager.CallOpts, id)
}

// GetLock is a free data retrieval call binding the contract method 0xd68f4dd1.
//
// Solidity: function getLock(uint256 i) view returns(address, address)
func (_BookManager *BookManagerCaller) GetLock(opts *bind.CallOpts, i *big.Int) (common.Address, common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getLock", i)

	if err != nil {
		return *new(common.Address), *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	out1 := *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return out0, out1, err

}

// GetLock is a free data retrieval call binding the contract method 0xd68f4dd1.
//
// Solidity: function getLock(uint256 i) view returns(address, address)
func (_BookManager *BookManagerSession) GetLock(i *big.Int) (common.Address, common.Address, error) {
	return _BookManager.Contract.GetLock(&_BookManager.CallOpts, i)
}

// GetLock is a free data retrieval call binding the contract method 0xd68f4dd1.
//
// Solidity: function getLock(uint256 i) view returns(address, address)
func (_BookManager *BookManagerCallerSession) GetLock(i *big.Int) (common.Address, common.Address, error) {
	return _BookManager.Contract.GetLock(&_BookManager.CallOpts, i)
}

// GetLockData is a free data retrieval call binding the contract method 0x4c02bf0b.
//
// Solidity: function getLockData() view returns(uint128, uint128)
func (_BookManager *BookManagerCaller) GetLockData(opts *bind.CallOpts) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getLockData")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetLockData is a free data retrieval call binding the contract method 0x4c02bf0b.
//
// Solidity: function getLockData() view returns(uint128, uint128)
func (_BookManager *BookManagerSession) GetLockData() (*big.Int, *big.Int, error) {
	return _BookManager.Contract.GetLockData(&_BookManager.CallOpts)
}

// GetLockData is a free data retrieval call binding the contract method 0x4c02bf0b.
//
// Solidity: function getLockData() view returns(uint128, uint128)
func (_BookManager *BookManagerCallerSession) GetLockData() (*big.Int, *big.Int, error) {
	return _BookManager.Contract.GetLockData(&_BookManager.CallOpts)
}

// GetOrder is a free data retrieval call binding the contract method 0xd09ef241.
//
// Solidity: function getOrder(uint256 id) view returns((address,uint64,uint64))
func (_BookManager *BookManagerCaller) GetOrder(opts *bind.CallOpts, id *big.Int) (IBookManagerOrderInfo, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "getOrder", id)

	if err != nil {
		return *new(IBookManagerOrderInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(IBookManagerOrderInfo)).(*IBookManagerOrderInfo)

	return out0, err

}

// GetOrder is a free data retrieval call binding the contract method 0xd09ef241.
//
// Solidity: function getOrder(uint256 id) view returns((address,uint64,uint64))
func (_BookManager *BookManagerSession) GetOrder(id *big.Int) (IBookManagerOrderInfo, error) {
	return _BookManager.Contract.GetOrder(&_BookManager.CallOpts, id)
}

// GetOrder is a free data retrieval call binding the contract method 0xd09ef241.
//
// Solidity: function getOrder(uint256 id) view returns((address,uint64,uint64))
func (_BookManager *BookManagerCallerSession) GetOrder(id *big.Int) (IBookManagerOrderInfo, error) {
	return _BookManager.Contract.GetOrder(&_BookManager.CallOpts, id)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_BookManager *BookManagerCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "isApprovedForAll", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_BookManager *BookManagerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _BookManager.Contract.IsApprovedForAll(&_BookManager.CallOpts, owner, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_BookManager *BookManagerCallerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _BookManager.Contract.IsApprovedForAll(&_BookManager.CallOpts, owner, operator)
}

// IsEmpty is a free data retrieval call binding the contract method 0xfcc8fc9b.
//
// Solidity: function isEmpty(uint192 id) view returns(bool)
func (_BookManager *BookManagerCaller) IsEmpty(opts *bind.CallOpts, id *big.Int) (bool, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "isEmpty", id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsEmpty is a free data retrieval call binding the contract method 0xfcc8fc9b.
//
// Solidity: function isEmpty(uint192 id) view returns(bool)
func (_BookManager *BookManagerSession) IsEmpty(id *big.Int) (bool, error) {
	return _BookManager.Contract.IsEmpty(&_BookManager.CallOpts, id)
}

// IsEmpty is a free data retrieval call binding the contract method 0xfcc8fc9b.
//
// Solidity: function isEmpty(uint192 id) view returns(bool)
func (_BookManager *BookManagerCallerSession) IsEmpty(id *big.Int) (bool, error) {
	return _BookManager.Contract.IsEmpty(&_BookManager.CallOpts, id)
}

// IsOpened is a free data retrieval call binding the contract method 0x55af6a32.
//
// Solidity: function isOpened(uint192 id) view returns(bool)
func (_BookManager *BookManagerCaller) IsOpened(opts *bind.CallOpts, id *big.Int) (bool, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "isOpened", id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOpened is a free data retrieval call binding the contract method 0x55af6a32.
//
// Solidity: function isOpened(uint192 id) view returns(bool)
func (_BookManager *BookManagerSession) IsOpened(id *big.Int) (bool, error) {
	return _BookManager.Contract.IsOpened(&_BookManager.CallOpts, id)
}

// IsOpened is a free data retrieval call binding the contract method 0x55af6a32.
//
// Solidity: function isOpened(uint192 id) view returns(bool)
func (_BookManager *BookManagerCallerSession) IsOpened(id *big.Int) (bool, error) {
	return _BookManager.Contract.IsOpened(&_BookManager.CallOpts, id)
}

// IsWhitelisted is a free data retrieval call binding the contract method 0x3af32abf.
//
// Solidity: function isWhitelisted(address provider) view returns(bool)
func (_BookManager *BookManagerCaller) IsWhitelisted(opts *bind.CallOpts, provider common.Address) (bool, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "isWhitelisted", provider)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWhitelisted is a free data retrieval call binding the contract method 0x3af32abf.
//
// Solidity: function isWhitelisted(address provider) view returns(bool)
func (_BookManager *BookManagerSession) IsWhitelisted(provider common.Address) (bool, error) {
	return _BookManager.Contract.IsWhitelisted(&_BookManager.CallOpts, provider)
}

// IsWhitelisted is a free data retrieval call binding the contract method 0x3af32abf.
//
// Solidity: function isWhitelisted(address provider) view returns(bool)
func (_BookManager *BookManagerCallerSession) IsWhitelisted(provider common.Address) (bool, error) {
	return _BookManager.Contract.IsWhitelisted(&_BookManager.CallOpts, provider)
}

// Load is a free data retrieval call binding the contract method 0xf0350799.
//
// Solidity: function load(bytes32 slot) view returns(bytes32 value)
func (_BookManager *BookManagerCaller) Load(opts *bind.CallOpts, slot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "load", slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Load is a free data retrieval call binding the contract method 0xf0350799.
//
// Solidity: function load(bytes32 slot) view returns(bytes32 value)
func (_BookManager *BookManagerSession) Load(slot [32]byte) ([32]byte, error) {
	return _BookManager.Contract.Load(&_BookManager.CallOpts, slot)
}

// Load is a free data retrieval call binding the contract method 0xf0350799.
//
// Solidity: function load(bytes32 slot) view returns(bytes32 value)
func (_BookManager *BookManagerCallerSession) Load(slot [32]byte) ([32]byte, error) {
	return _BookManager.Contract.Load(&_BookManager.CallOpts, slot)
}

// Load0 is a free data retrieval call binding the contract method 0xf86a11b3.
//
// Solidity: function load(bytes32 startSlot, uint256 nSlot) view returns(bytes value)
func (_BookManager *BookManagerCaller) Load0(opts *bind.CallOpts, startSlot [32]byte, nSlot *big.Int) ([]byte, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "load0", startSlot, nSlot)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Load0 is a free data retrieval call binding the contract method 0xf86a11b3.
//
// Solidity: function load(bytes32 startSlot, uint256 nSlot) view returns(bytes value)
func (_BookManager *BookManagerSession) Load0(startSlot [32]byte, nSlot *big.Int) ([]byte, error) {
	return _BookManager.Contract.Load0(&_BookManager.CallOpts, startSlot, nSlot)
}

// Load0 is a free data retrieval call binding the contract method 0xf86a11b3.
//
// Solidity: function load(bytes32 startSlot, uint256 nSlot) view returns(bytes value)
func (_BookManager *BookManagerCallerSession) Load0(startSlot [32]byte, nSlot *big.Int) ([]byte, error) {
	return _BookManager.Contract.Load0(&_BookManager.CallOpts, startSlot, nSlot)
}

// MaxLessThan is a free data retrieval call binding the contract method 0xa179dadc.
//
// Solidity: function maxLessThan(uint192 id, int24 tick) view returns(int24)
func (_BookManager *BookManagerCaller) MaxLessThan(opts *bind.CallOpts, id *big.Int, tick *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "maxLessThan", id, tick)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxLessThan is a free data retrieval call binding the contract method 0xa179dadc.
//
// Solidity: function maxLessThan(uint192 id, int24 tick) view returns(int24)
func (_BookManager *BookManagerSession) MaxLessThan(id *big.Int, tick *big.Int) (*big.Int, error) {
	return _BookManager.Contract.MaxLessThan(&_BookManager.CallOpts, id, tick)
}

// MaxLessThan is a free data retrieval call binding the contract method 0xa179dadc.
//
// Solidity: function maxLessThan(uint192 id, int24 tick) view returns(int24)
func (_BookManager *BookManagerCallerSession) MaxLessThan(id *big.Int, tick *big.Int) (*big.Int, error) {
	return _BookManager.Contract.MaxLessThan(&_BookManager.CallOpts, id, tick)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_BookManager *BookManagerCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_BookManager *BookManagerSession) Name() (string, error) {
	return _BookManager.Contract.Name(&_BookManager.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_BookManager *BookManagerCallerSession) Name() (string, error) {
	return _BookManager.Contract.Name(&_BookManager.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x141a468c.
//
// Solidity: function nonces(uint256 id) view returns(uint256)
func (_BookManager *BookManagerCaller) Nonces(opts *bind.CallOpts, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "nonces", id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x141a468c.
//
// Solidity: function nonces(uint256 id) view returns(uint256)
func (_BookManager *BookManagerSession) Nonces(id *big.Int) (*big.Int, error) {
	return _BookManager.Contract.Nonces(&_BookManager.CallOpts, id)
}

// Nonces is a free data retrieval call binding the contract method 0x141a468c.
//
// Solidity: function nonces(uint256 id) view returns(uint256)
func (_BookManager *BookManagerCallerSession) Nonces(id *big.Int) (*big.Int, error) {
	return _BookManager.Contract.Nonces(&_BookManager.CallOpts, id)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BookManager *BookManagerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BookManager *BookManagerSession) Owner() (common.Address, error) {
	return _BookManager.Contract.Owner(&_BookManager.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BookManager *BookManagerCallerSession) Owner() (common.Address, error) {
	return _BookManager.Contract.Owner(&_BookManager.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerCaller) OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "ownerOf", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _BookManager.Contract.OwnerOf(&_BookManager.CallOpts, tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_BookManager *BookManagerCallerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _BookManager.Contract.OwnerOf(&_BookManager.CallOpts, tokenId)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_BookManager *BookManagerCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_BookManager *BookManagerSession) PendingOwner() (common.Address, error) {
	return _BookManager.Contract.PendingOwner(&_BookManager.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_BookManager *BookManagerCallerSession) PendingOwner() (common.Address, error) {
	return _BookManager.Contract.PendingOwner(&_BookManager.CallOpts)
}

// ReservesOf is a free data retrieval call binding the contract method 0x93c85a21.
//
// Solidity: function reservesOf(address currency) view returns(uint256)
func (_BookManager *BookManagerCaller) ReservesOf(opts *bind.CallOpts, currency common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "reservesOf", currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReservesOf is a free data retrieval call binding the contract method 0x93c85a21.
//
// Solidity: function reservesOf(address currency) view returns(uint256)
func (_BookManager *BookManagerSession) ReservesOf(currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.ReservesOf(&_BookManager.CallOpts, currency)
}

// ReservesOf is a free data retrieval call binding the contract method 0x93c85a21.
//
// Solidity: function reservesOf(address currency) view returns(uint256)
func (_BookManager *BookManagerCallerSession) ReservesOf(currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.ReservesOf(&_BookManager.CallOpts, currency)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BookManager *BookManagerCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BookManager *BookManagerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BookManager.Contract.SupportsInterface(&_BookManager.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BookManager *BookManagerCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BookManager.Contract.SupportsInterface(&_BookManager.CallOpts, interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_BookManager *BookManagerCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_BookManager *BookManagerSession) Symbol() (string, error) {
	return _BookManager.Contract.Symbol(&_BookManager.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_BookManager *BookManagerCallerSession) Symbol() (string, error) {
	return _BookManager.Contract.Symbol(&_BookManager.CallOpts)
}

// TokenOwed is a free data retrieval call binding the contract method 0x3e547b06.
//
// Solidity: function tokenOwed(address provider, address currency) view returns(uint256 amount)
func (_BookManager *BookManagerCaller) TokenOwed(opts *bind.CallOpts, provider common.Address, currency common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "tokenOwed", provider, currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenOwed is a free data retrieval call binding the contract method 0x3e547b06.
//
// Solidity: function tokenOwed(address provider, address currency) view returns(uint256 amount)
func (_BookManager *BookManagerSession) TokenOwed(provider common.Address, currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.TokenOwed(&_BookManager.CallOpts, provider, currency)
}

// TokenOwed is a free data retrieval call binding the contract method 0x3e547b06.
//
// Solidity: function tokenOwed(address provider, address currency) view returns(uint256 amount)
func (_BookManager *BookManagerCallerSession) TokenOwed(provider common.Address, currency common.Address) (*big.Int, error) {
	return _BookManager.Contract.TokenOwed(&_BookManager.CallOpts, provider, currency)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_BookManager *BookManagerCaller) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _BookManager.contract.Call(opts, &out, "tokenURI", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_BookManager *BookManagerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _BookManager.Contract.TokenURI(&_BookManager.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_BookManager *BookManagerCallerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _BookManager.Contract.TokenURI(&_BookManager.CallOpts, tokenId)
}

// BookManagerApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the BookManager contract.
type BookManagerApprovalIterator struct {
	Event *BookManagerApproval // Event containing the contract specifics and raw log

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
func (it *BookManagerApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerApproval)
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
		it.Event = new(BookManagerApproval)
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
func (it *BookManagerApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerApproval represents a Approval event raised by the BookManager contract.
type BookManagerApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, approved []common.Address, tokenId []*big.Int) (*BookManagerApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerApprovalIterator{contract: _BookManager.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *BookManagerApproval, owner []common.Address, approved []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerApproval)
				if err := _BookManager.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) ParseApproval(log types.Log) (*BookManagerApproval, error) {
	event := new(BookManagerApproval)
	if err := _BookManager.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the BookManager contract.
type BookManagerApprovalForAllIterator struct {
	Event *BookManagerApprovalForAll // Event containing the contract specifics and raw log

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
func (it *BookManagerApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerApprovalForAll)
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
		it.Event = new(BookManagerApprovalForAll)
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
func (it *BookManagerApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerApprovalForAll represents a ApprovalForAll event raised by the BookManager contract.
type BookManagerApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_BookManager *BookManagerFilterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*BookManagerApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerApprovalForAllIterator{contract: _BookManager.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_BookManager *BookManagerFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *BookManagerApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerApprovalForAll)
				if err := _BookManager.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_BookManager *BookManagerFilterer) ParseApprovalForAll(log types.Log) (*BookManagerApprovalForAll, error) {
	event := new(BookManagerApprovalForAll)
	if err := _BookManager.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerCancelIterator is returned from FilterCancel and is used to iterate over the raw logs and unpacked data for Cancel events raised by the BookManager contract.
type BookManagerCancelIterator struct {
	Event *BookManagerCancel // Event containing the contract specifics and raw log

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
func (it *BookManagerCancelIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerCancel)
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
		it.Event = new(BookManagerCancel)
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
func (it *BookManagerCancelIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerCancelIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerCancel represents a Cancel event raised by the BookManager contract.
type BookManagerCancel struct {
	OrderId *big.Int
	Unit    uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterCancel is a free log retrieval operation binding the contract event 0x0c6ba7ef5064094c17cce013aa4c617a23e2582f867774d07a5931de43b85d72.
//
// Solidity: event Cancel(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) FilterCancel(opts *bind.FilterOpts, orderId []*big.Int) (*BookManagerCancelIterator, error) {

	var orderIdRule []interface{}
	for _, orderIdItem := range orderId {
		orderIdRule = append(orderIdRule, orderIdItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Cancel", orderIdRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerCancelIterator{contract: _BookManager.contract, event: "Cancel", logs: logs, sub: sub}, nil
}

// WatchCancel is a free log subscription operation binding the contract event 0x0c6ba7ef5064094c17cce013aa4c617a23e2582f867774d07a5931de43b85d72.
//
// Solidity: event Cancel(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) WatchCancel(opts *bind.WatchOpts, sink chan<- *BookManagerCancel, orderId []*big.Int) (event.Subscription, error) {

	var orderIdRule []interface{}
	for _, orderIdItem := range orderId {
		orderIdRule = append(orderIdRule, orderIdItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Cancel", orderIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerCancel)
				if err := _BookManager.contract.UnpackLog(event, "Cancel", log); err != nil {
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

// ParseCancel is a log parse operation binding the contract event 0x0c6ba7ef5064094c17cce013aa4c617a23e2582f867774d07a5931de43b85d72.
//
// Solidity: event Cancel(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) ParseCancel(log types.Log) (*BookManagerCancel, error) {
	event := new(BookManagerCancel)
	if err := _BookManager.contract.UnpackLog(event, "Cancel", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerClaimIterator is returned from FilterClaim and is used to iterate over the raw logs and unpacked data for Claim events raised by the BookManager contract.
type BookManagerClaimIterator struct {
	Event *BookManagerClaim // Event containing the contract specifics and raw log

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
func (it *BookManagerClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerClaim)
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
		it.Event = new(BookManagerClaim)
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
func (it *BookManagerClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerClaim represents a Claim event raised by the BookManager contract.
type BookManagerClaim struct {
	OrderId *big.Int
	Unit    uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterClaim is a free log retrieval operation binding the contract event 0xfc7df80a30ee916cc040221cf6fcfb3c6dc994b3fa4c4ab23e8a0f134de5c0c0.
//
// Solidity: event Claim(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) FilterClaim(opts *bind.FilterOpts, orderId []*big.Int) (*BookManagerClaimIterator, error) {

	var orderIdRule []interface{}
	for _, orderIdItem := range orderId {
		orderIdRule = append(orderIdRule, orderIdItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Claim", orderIdRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerClaimIterator{contract: _BookManager.contract, event: "Claim", logs: logs, sub: sub}, nil
}

// WatchClaim is a free log subscription operation binding the contract event 0xfc7df80a30ee916cc040221cf6fcfb3c6dc994b3fa4c4ab23e8a0f134de5c0c0.
//
// Solidity: event Claim(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) WatchClaim(opts *bind.WatchOpts, sink chan<- *BookManagerClaim, orderId []*big.Int) (event.Subscription, error) {

	var orderIdRule []interface{}
	for _, orderIdItem := range orderId {
		orderIdRule = append(orderIdRule, orderIdItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Claim", orderIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerClaim)
				if err := _BookManager.contract.UnpackLog(event, "Claim", log); err != nil {
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

// ParseClaim is a log parse operation binding the contract event 0xfc7df80a30ee916cc040221cf6fcfb3c6dc994b3fa4c4ab23e8a0f134de5c0c0.
//
// Solidity: event Claim(uint256 indexed orderId, uint64 unit)
func (_BookManager *BookManagerFilterer) ParseClaim(log types.Log) (*BookManagerClaim, error) {
	event := new(BookManagerClaim)
	if err := _BookManager.contract.UnpackLog(event, "Claim", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerCollectIterator is returned from FilterCollect and is used to iterate over the raw logs and unpacked data for Collect events raised by the BookManager contract.
type BookManagerCollectIterator struct {
	Event *BookManagerCollect // Event containing the contract specifics and raw log

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
func (it *BookManagerCollectIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerCollect)
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
		it.Event = new(BookManagerCollect)
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
func (it *BookManagerCollectIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerCollectIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerCollect represents a Collect event raised by the BookManager contract.
type BookManagerCollect struct {
	Provider  common.Address
	Recipient common.Address
	Currency  common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollect is a free log retrieval operation binding the contract event 0x1c4f94f28cc9152354d4b98b8614b28c6c828a98d88228fa9577c7b9475e120c.
//
// Solidity: event Collect(address indexed provider, address indexed recipient, address indexed currency, uint256 amount)
func (_BookManager *BookManagerFilterer) FilterCollect(opts *bind.FilterOpts, provider []common.Address, recipient []common.Address, currency []common.Address) (*BookManagerCollectIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var currencyRule []interface{}
	for _, currencyItem := range currency {
		currencyRule = append(currencyRule, currencyItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Collect", providerRule, recipientRule, currencyRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerCollectIterator{contract: _BookManager.contract, event: "Collect", logs: logs, sub: sub}, nil
}

// WatchCollect is a free log subscription operation binding the contract event 0x1c4f94f28cc9152354d4b98b8614b28c6c828a98d88228fa9577c7b9475e120c.
//
// Solidity: event Collect(address indexed provider, address indexed recipient, address indexed currency, uint256 amount)
func (_BookManager *BookManagerFilterer) WatchCollect(opts *bind.WatchOpts, sink chan<- *BookManagerCollect, provider []common.Address, recipient []common.Address, currency []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var currencyRule []interface{}
	for _, currencyItem := range currency {
		currencyRule = append(currencyRule, currencyItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Collect", providerRule, recipientRule, currencyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerCollect)
				if err := _BookManager.contract.UnpackLog(event, "Collect", log); err != nil {
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

// ParseCollect is a log parse operation binding the contract event 0x1c4f94f28cc9152354d4b98b8614b28c6c828a98d88228fa9577c7b9475e120c.
//
// Solidity: event Collect(address indexed provider, address indexed recipient, address indexed currency, uint256 amount)
func (_BookManager *BookManagerFilterer) ParseCollect(log types.Log) (*BookManagerCollect, error) {
	event := new(BookManagerCollect)
	if err := _BookManager.contract.UnpackLog(event, "Collect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerDelistIterator is returned from FilterDelist and is used to iterate over the raw logs and unpacked data for Delist events raised by the BookManager contract.
type BookManagerDelistIterator struct {
	Event *BookManagerDelist // Event containing the contract specifics and raw log

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
func (it *BookManagerDelistIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerDelist)
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
		it.Event = new(BookManagerDelist)
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
func (it *BookManagerDelistIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerDelistIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerDelist represents a Delist event raised by the BookManager contract.
type BookManagerDelist struct {
	Provider common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterDelist is a free log retrieval operation binding the contract event 0x88f58aa68e1f754fecfec41a6758d18d4a53fa15d4e206fd54bbdfe7a9e98da7.
//
// Solidity: event Delist(address indexed provider)
func (_BookManager *BookManagerFilterer) FilterDelist(opts *bind.FilterOpts, provider []common.Address) (*BookManagerDelistIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Delist", providerRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerDelistIterator{contract: _BookManager.contract, event: "Delist", logs: logs, sub: sub}, nil
}

// WatchDelist is a free log subscription operation binding the contract event 0x88f58aa68e1f754fecfec41a6758d18d4a53fa15d4e206fd54bbdfe7a9e98da7.
//
// Solidity: event Delist(address indexed provider)
func (_BookManager *BookManagerFilterer) WatchDelist(opts *bind.WatchOpts, sink chan<- *BookManagerDelist, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Delist", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerDelist)
				if err := _BookManager.contract.UnpackLog(event, "Delist", log); err != nil {
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

// ParseDelist is a log parse operation binding the contract event 0x88f58aa68e1f754fecfec41a6758d18d4a53fa15d4e206fd54bbdfe7a9e98da7.
//
// Solidity: event Delist(address indexed provider)
func (_BookManager *BookManagerFilterer) ParseDelist(log types.Log) (*BookManagerDelist, error) {
	event := new(BookManagerDelist)
	if err := _BookManager.contract.UnpackLog(event, "Delist", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the BookManager contract.
type BookManagerEIP712DomainChangedIterator struct {
	Event *BookManagerEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *BookManagerEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerEIP712DomainChanged)
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
		it.Event = new(BookManagerEIP712DomainChanged)
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
func (it *BookManagerEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerEIP712DomainChanged represents a EIP712DomainChanged event raised by the BookManager contract.
type BookManagerEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_BookManager *BookManagerFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*BookManagerEIP712DomainChangedIterator, error) {

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &BookManagerEIP712DomainChangedIterator{contract: _BookManager.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_BookManager *BookManagerFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *BookManagerEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerEIP712DomainChanged)
				if err := _BookManager.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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

// ParseEIP712DomainChanged is a log parse operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_BookManager *BookManagerFilterer) ParseEIP712DomainChanged(log types.Log) (*BookManagerEIP712DomainChanged, error) {
	event := new(BookManagerEIP712DomainChanged)
	if err := _BookManager.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerMakeIterator is returned from FilterMake and is used to iterate over the raw logs and unpacked data for Make events raised by the BookManager contract.
type BookManagerMakeIterator struct {
	Event *BookManagerMake // Event containing the contract specifics and raw log

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
func (it *BookManagerMakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerMake)
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
		it.Event = new(BookManagerMake)
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
func (it *BookManagerMakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerMakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerMake represents a Make event raised by the BookManager contract.
type BookManagerMake struct {
	BookId     *big.Int
	User       common.Address
	Tick       *big.Int
	OrderIndex *big.Int
	Unit       uint64
	Provider   common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterMake is a free log retrieval operation binding the contract event 0x251db4df45fa692f68b4e3f072919384c5b71995c71bf22888385168930fd22a.
//
// Solidity: event Make(uint192 indexed bookId, address indexed user, int24 tick, uint256 orderIndex, uint64 unit, address provider)
func (_BookManager *BookManagerFilterer) FilterMake(opts *bind.FilterOpts, bookId []*big.Int, user []common.Address) (*BookManagerMakeIterator, error) {

	var bookIdRule []interface{}
	for _, bookIdItem := range bookId {
		bookIdRule = append(bookIdRule, bookIdItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Make", bookIdRule, userRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerMakeIterator{contract: _BookManager.contract, event: "Make", logs: logs, sub: sub}, nil
}

// WatchMake is a free log subscription operation binding the contract event 0x251db4df45fa692f68b4e3f072919384c5b71995c71bf22888385168930fd22a.
//
// Solidity: event Make(uint192 indexed bookId, address indexed user, int24 tick, uint256 orderIndex, uint64 unit, address provider)
func (_BookManager *BookManagerFilterer) WatchMake(opts *bind.WatchOpts, sink chan<- *BookManagerMake, bookId []*big.Int, user []common.Address) (event.Subscription, error) {

	var bookIdRule []interface{}
	for _, bookIdItem := range bookId {
		bookIdRule = append(bookIdRule, bookIdItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Make", bookIdRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerMake)
				if err := _BookManager.contract.UnpackLog(event, "Make", log); err != nil {
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

// ParseMake is a log parse operation binding the contract event 0x251db4df45fa692f68b4e3f072919384c5b71995c71bf22888385168930fd22a.
//
// Solidity: event Make(uint192 indexed bookId, address indexed user, int24 tick, uint256 orderIndex, uint64 unit, address provider)
func (_BookManager *BookManagerFilterer) ParseMake(log types.Log) (*BookManagerMake, error) {
	event := new(BookManagerMake)
	if err := _BookManager.contract.UnpackLog(event, "Make", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerOpenIterator is returned from FilterOpen and is used to iterate over the raw logs and unpacked data for Open events raised by the BookManager contract.
type BookManagerOpenIterator struct {
	Event *BookManagerOpen // Event containing the contract specifics and raw log

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
func (it *BookManagerOpenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerOpen)
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
		it.Event = new(BookManagerOpen)
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
func (it *BookManagerOpenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerOpenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerOpen represents a Open event raised by the BookManager contract.
type BookManagerOpen struct {
	Id          *big.Int
	Base        common.Address
	Quote       common.Address
	UnitSize    uint64
	MakerPolicy *big.Int
	TakerPolicy *big.Int
	Hooks       common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterOpen is a free log retrieval operation binding the contract event 0x803427d75ce3214f82dc7aa4910635170a6655e2c1663dc03429dd04100cba5a.
//
// Solidity: event Open(uint192 indexed id, address indexed base, address indexed quote, uint64 unitSize, uint24 makerPolicy, uint24 takerPolicy, address hooks)
func (_BookManager *BookManagerFilterer) FilterOpen(opts *bind.FilterOpts, id []*big.Int, base []common.Address, quote []common.Address) (*BookManagerOpenIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var baseRule []interface{}
	for _, baseItem := range base {
		baseRule = append(baseRule, baseItem)
	}
	var quoteRule []interface{}
	for _, quoteItem := range quote {
		quoteRule = append(quoteRule, quoteItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Open", idRule, baseRule, quoteRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerOpenIterator{contract: _BookManager.contract, event: "Open", logs: logs, sub: sub}, nil
}

// WatchOpen is a free log subscription operation binding the contract event 0x803427d75ce3214f82dc7aa4910635170a6655e2c1663dc03429dd04100cba5a.
//
// Solidity: event Open(uint192 indexed id, address indexed base, address indexed quote, uint64 unitSize, uint24 makerPolicy, uint24 takerPolicy, address hooks)
func (_BookManager *BookManagerFilterer) WatchOpen(opts *bind.WatchOpts, sink chan<- *BookManagerOpen, id []*big.Int, base []common.Address, quote []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var baseRule []interface{}
	for _, baseItem := range base {
		baseRule = append(baseRule, baseItem)
	}
	var quoteRule []interface{}
	for _, quoteItem := range quote {
		quoteRule = append(quoteRule, quoteItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Open", idRule, baseRule, quoteRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerOpen)
				if err := _BookManager.contract.UnpackLog(event, "Open", log); err != nil {
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

// ParseOpen is a log parse operation binding the contract event 0x803427d75ce3214f82dc7aa4910635170a6655e2c1663dc03429dd04100cba5a.
//
// Solidity: event Open(uint192 indexed id, address indexed base, address indexed quote, uint64 unitSize, uint24 makerPolicy, uint24 takerPolicy, address hooks)
func (_BookManager *BookManagerFilterer) ParseOpen(log types.Log) (*BookManagerOpen, error) {
	event := new(BookManagerOpen)
	if err := _BookManager.contract.UnpackLog(event, "Open", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the BookManager contract.
type BookManagerOwnershipTransferStartedIterator struct {
	Event *BookManagerOwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *BookManagerOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerOwnershipTransferStarted)
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
		it.Event = new(BookManagerOwnershipTransferStarted)
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
func (it *BookManagerOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the BookManager contract.
type BookManagerOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BookManagerOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerOwnershipTransferStartedIterator{contract: _BookManager.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *BookManagerOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerOwnershipTransferStarted)
				if err := _BookManager.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) ParseOwnershipTransferStarted(log types.Log) (*BookManagerOwnershipTransferStarted, error) {
	event := new(BookManagerOwnershipTransferStarted)
	if err := _BookManager.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BookManager contract.
type BookManagerOwnershipTransferredIterator struct {
	Event *BookManagerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *BookManagerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerOwnershipTransferred)
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
		it.Event = new(BookManagerOwnershipTransferred)
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
func (it *BookManagerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerOwnershipTransferred represents a OwnershipTransferred event raised by the BookManager contract.
type BookManagerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BookManagerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerOwnershipTransferredIterator{contract: _BookManager.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BookManagerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerOwnershipTransferred)
				if err := _BookManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BookManager *BookManagerFilterer) ParseOwnershipTransferred(log types.Log) (*BookManagerOwnershipTransferred, error) {
	event := new(BookManagerOwnershipTransferred)
	if err := _BookManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerSetDefaultProviderIterator is returned from FilterSetDefaultProvider and is used to iterate over the raw logs and unpacked data for SetDefaultProvider events raised by the BookManager contract.
type BookManagerSetDefaultProviderIterator struct {
	Event *BookManagerSetDefaultProvider // Event containing the contract specifics and raw log

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
func (it *BookManagerSetDefaultProviderIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerSetDefaultProvider)
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
		it.Event = new(BookManagerSetDefaultProvider)
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
func (it *BookManagerSetDefaultProviderIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerSetDefaultProviderIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerSetDefaultProvider represents a SetDefaultProvider event raised by the BookManager contract.
type BookManagerSetDefaultProvider struct {
	NewDefaultProvider common.Address
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterSetDefaultProvider is a free log retrieval operation binding the contract event 0xef673bbfc2ac7e4d4b810bffda0b15a1f2b48c2aa4d178d3fca87d0d1f337062.
//
// Solidity: event SetDefaultProvider(address indexed newDefaultProvider)
func (_BookManager *BookManagerFilterer) FilterSetDefaultProvider(opts *bind.FilterOpts, newDefaultProvider []common.Address) (*BookManagerSetDefaultProviderIterator, error) {

	var newDefaultProviderRule []interface{}
	for _, newDefaultProviderItem := range newDefaultProvider {
		newDefaultProviderRule = append(newDefaultProviderRule, newDefaultProviderItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "SetDefaultProvider", newDefaultProviderRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerSetDefaultProviderIterator{contract: _BookManager.contract, event: "SetDefaultProvider", logs: logs, sub: sub}, nil
}

// WatchSetDefaultProvider is a free log subscription operation binding the contract event 0xef673bbfc2ac7e4d4b810bffda0b15a1f2b48c2aa4d178d3fca87d0d1f337062.
//
// Solidity: event SetDefaultProvider(address indexed newDefaultProvider)
func (_BookManager *BookManagerFilterer) WatchSetDefaultProvider(opts *bind.WatchOpts, sink chan<- *BookManagerSetDefaultProvider, newDefaultProvider []common.Address) (event.Subscription, error) {

	var newDefaultProviderRule []interface{}
	for _, newDefaultProviderItem := range newDefaultProvider {
		newDefaultProviderRule = append(newDefaultProviderRule, newDefaultProviderItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "SetDefaultProvider", newDefaultProviderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerSetDefaultProvider)
				if err := _BookManager.contract.UnpackLog(event, "SetDefaultProvider", log); err != nil {
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

// ParseSetDefaultProvider is a log parse operation binding the contract event 0xef673bbfc2ac7e4d4b810bffda0b15a1f2b48c2aa4d178d3fca87d0d1f337062.
//
// Solidity: event SetDefaultProvider(address indexed newDefaultProvider)
func (_BookManager *BookManagerFilterer) ParseSetDefaultProvider(log types.Log) (*BookManagerSetDefaultProvider, error) {
	event := new(BookManagerSetDefaultProvider)
	if err := _BookManager.contract.UnpackLog(event, "SetDefaultProvider", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerTakeIterator is returned from FilterTake and is used to iterate over the raw logs and unpacked data for Take events raised by the BookManager contract.
type BookManagerTakeIterator struct {
	Event *BookManagerTake // Event containing the contract specifics and raw log

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
func (it *BookManagerTakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerTake)
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
		it.Event = new(BookManagerTake)
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
func (it *BookManagerTakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerTakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerTake represents a Take event raised by the BookManager contract.
type BookManagerTake struct {
	BookId *big.Int
	User   common.Address
	Tick   *big.Int
	Unit   uint64
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTake is a free log retrieval operation binding the contract event 0xc4c20b9c4a5ada3b01b7a391a08dd81a1be01dd8ef63170dd9da44ecee3db11b.
//
// Solidity: event Take(uint192 indexed bookId, address indexed user, int24 tick, uint64 unit)
func (_BookManager *BookManagerFilterer) FilterTake(opts *bind.FilterOpts, bookId []*big.Int, user []common.Address) (*BookManagerTakeIterator, error) {

	var bookIdRule []interface{}
	for _, bookIdItem := range bookId {
		bookIdRule = append(bookIdRule, bookIdItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Take", bookIdRule, userRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerTakeIterator{contract: _BookManager.contract, event: "Take", logs: logs, sub: sub}, nil
}

// WatchTake is a free log subscription operation binding the contract event 0xc4c20b9c4a5ada3b01b7a391a08dd81a1be01dd8ef63170dd9da44ecee3db11b.
//
// Solidity: event Take(uint192 indexed bookId, address indexed user, int24 tick, uint64 unit)
func (_BookManager *BookManagerFilterer) WatchTake(opts *bind.WatchOpts, sink chan<- *BookManagerTake, bookId []*big.Int, user []common.Address) (event.Subscription, error) {

	var bookIdRule []interface{}
	for _, bookIdItem := range bookId {
		bookIdRule = append(bookIdRule, bookIdItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Take", bookIdRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerTake)
				if err := _BookManager.contract.UnpackLog(event, "Take", log); err != nil {
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

// ParseTake is a log parse operation binding the contract event 0xc4c20b9c4a5ada3b01b7a391a08dd81a1be01dd8ef63170dd9da44ecee3db11b.
//
// Solidity: event Take(uint192 indexed bookId, address indexed user, int24 tick, uint64 unit)
func (_BookManager *BookManagerFilterer) ParseTake(log types.Log) (*BookManagerTake, error) {
	event := new(BookManagerTake)
	if err := _BookManager.contract.UnpackLog(event, "Take", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the BookManager contract.
type BookManagerTransferIterator struct {
	Event *BookManagerTransfer // Event containing the contract specifics and raw log

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
func (it *BookManagerTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerTransfer)
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
		it.Event = new(BookManagerTransfer)
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
func (it *BookManagerTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerTransfer represents a Transfer event raised by the BookManager contract.
type BookManagerTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, tokenId []*big.Int) (*BookManagerTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerTransferIterator{contract: _BookManager.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *BookManagerTransfer, from []common.Address, to []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerTransfer)
				if err := _BookManager.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_BookManager *BookManagerFilterer) ParseTransfer(log types.Log) (*BookManagerTransfer, error) {
	event := new(BookManagerTransfer)
	if err := _BookManager.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BookManagerWhitelistIterator is returned from FilterWhitelist and is used to iterate over the raw logs and unpacked data for Whitelist events raised by the BookManager contract.
type BookManagerWhitelistIterator struct {
	Event *BookManagerWhitelist // Event containing the contract specifics and raw log

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
func (it *BookManagerWhitelistIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BookManagerWhitelist)
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
		it.Event = new(BookManagerWhitelist)
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
func (it *BookManagerWhitelistIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BookManagerWhitelistIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BookManagerWhitelist represents a Whitelist event raised by the BookManager contract.
type BookManagerWhitelist struct {
	Provider common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterWhitelist is a free log retrieval operation binding the contract event 0xeb73900b98b6a3e2b8b01708fe544760cf570d21e7fbe5225f24e48b5b2b432e.
//
// Solidity: event Whitelist(address indexed provider)
func (_BookManager *BookManagerFilterer) FilterWhitelist(opts *bind.FilterOpts, provider []common.Address) (*BookManagerWhitelistIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BookManager.contract.FilterLogs(opts, "Whitelist", providerRule)
	if err != nil {
		return nil, err
	}
	return &BookManagerWhitelistIterator{contract: _BookManager.contract, event: "Whitelist", logs: logs, sub: sub}, nil
}

// WatchWhitelist is a free log subscription operation binding the contract event 0xeb73900b98b6a3e2b8b01708fe544760cf570d21e7fbe5225f24e48b5b2b432e.
//
// Solidity: event Whitelist(address indexed provider)
func (_BookManager *BookManagerFilterer) WatchWhitelist(opts *bind.WatchOpts, sink chan<- *BookManagerWhitelist, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BookManager.contract.WatchLogs(opts, "Whitelist", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BookManagerWhitelist)
				if err := _BookManager.contract.UnpackLog(event, "Whitelist", log); err != nil {
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

// ParseWhitelist is a log parse operation binding the contract event 0xeb73900b98b6a3e2b8b01708fe544760cf570d21e7fbe5225f24e48b5b2b432e.
//
// Solidity: event Whitelist(address indexed provider)
func (_BookManager *BookManagerFilterer) ParseWhitelist(log types.Log) (*BookManagerWhitelist, error) {
	event := new(BookManagerWhitelist)
	if err := _BookManager.contract.UnpackLog(event, "Whitelist", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
