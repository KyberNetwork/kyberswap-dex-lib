// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package clanker

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

// IClankerDeploymentConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerDeploymentConfig struct {
	TokenConfig      IClankerTokenConfig
	PoolConfig       IClankerPoolConfig
	LockerConfig     IClankerLockerConfig
	MevModuleConfig  IClankerMevModuleConfig
	ExtensionConfigs []IClankerExtensionConfig
}

// IClankerDeploymentInfo is an auto generated low-level Go binding around an user-defined struct.
type IClankerDeploymentInfo struct {
	Token      common.Address
	Hook       common.Address
	Locker     common.Address
	Extensions []common.Address
}

// IClankerExtensionConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerExtensionConfig struct {
	Extension     common.Address
	MsgValue      *big.Int
	ExtensionBps  uint16
	ExtensionData []byte
}

// IClankerLockerConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerLockerConfig struct {
	Locker           common.Address
	RewardAdmins     []common.Address
	RewardRecipients []common.Address
	RewardBps        []uint16
	TickLower        []*big.Int
	TickUpper        []*big.Int
	PositionBps      []uint16
	LockerData       []byte
}

// IClankerMevModuleConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerMevModuleConfig struct {
	MevModule     common.Address
	MevModuleData []byte
}

// IClankerPoolConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerPoolConfig struct {
	Hook                  common.Address
	PairedToken           common.Address
	TickIfToken0IsClanker *big.Int
	TickSpacing           *big.Int
	PoolData              []byte
}

// IClankerTokenConfig is an auto generated low-level Go binding around an user-defined struct.
type IClankerTokenConfig struct {
	TokenAdmin         common.Address
	Name               string
	Symbol             string
	Salt               [32]byte
	Image              string
	Metadata           string
	Context            string
	OriginatingChainId *big.Int
}

// ClankerMetaData contains all meta data concerning the Clanker contract.
var ClankerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"Deprecated\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExtensionMsgValueMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExtensionNotEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HookNotEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidExtension\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHook\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidLocker\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidMevModule\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LockerNotEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxExtensionBpsExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxExtensionsExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MevModuleNotEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyNonOriginatingChains\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyOriginatingChain\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardReentrantCall\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"SafeERC20FailedOperation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TeamFeeRecipientNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"ClaimTeamFees\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"extension\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"extensionSupply\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"msgValue\",\"type\":\"uint256\"}],\"name\":\"ExtensionTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"SetAdmin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"deprecated\",\"type\":\"bool\"}],\"name\":\"SetDeprecated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"extension\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"SetExtension\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"SetHook\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"SetLocker\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"mevModule\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"SetMevModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldTeamFeeRecipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newTeamFeeRecipient\",\"type\":\"address\"}],\"name\":\"SetTeamFeeRecipient\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"msgSender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"tokenAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tokenImage\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tokenName\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tokenSymbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tokenMetadata\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tokenContext\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"startingTick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"poolHook\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"mevModule\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"extensionsSupply\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address[]\",\"name\":\"extensions\",\"type\":\"address[]\"}],\"name\":\"TokenCreated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BPS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_EXTENSIONS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_EXTENSION_BPS\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TOKEN_SUPPLY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"admins\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"claimTeamFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenAdmin\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"image\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"metadata\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"context\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"originatingChainId\",\"type\":\"uint256\"}],\"internalType\":\"structIClanker.TokenConfig\",\"name\":\"tokenConfig\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickIfToken0IsClanker\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"bytes\",\"name\":\"poolData\",\"type\":\"bytes\"}],\"internalType\":\"structIClanker.PoolConfig\",\"name\":\"poolConfig\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"rewardAdmins\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"rewardRecipients\",\"type\":\"address[]\"},{\"internalType\":\"uint16[]\",\"name\":\"rewardBps\",\"type\":\"uint16[]\"},{\"internalType\":\"int24[]\",\"name\":\"tickLower\",\"type\":\"int24[]\"},{\"internalType\":\"int24[]\",\"name\":\"tickUpper\",\"type\":\"int24[]\"},{\"internalType\":\"uint16[]\",\"name\":\"positionBps\",\"type\":\"uint16[]\"},{\"internalType\":\"bytes\",\"name\":\"lockerData\",\"type\":\"bytes\"}],\"internalType\":\"structIClanker.LockerConfig\",\"name\":\"lockerConfig\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"mevModule\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"mevModuleData\",\"type\":\"bytes\"}],\"internalType\":\"structIClanker.MevModuleConfig\",\"name\":\"mevModuleConfig\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"extension\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"msgValue\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"extensionBps\",\"type\":\"uint16\"},{\"internalType\":\"bytes\",\"name\":\"extensionData\",\"type\":\"bytes\"}],\"internalType\":\"structIClanker.ExtensionConfig[]\",\"name\":\"extensionConfigs\",\"type\":\"tuple[]\"}],\"internalType\":\"structIClanker.DeploymentConfig\",\"name\":\"deploymentConfig\",\"type\":\"tuple\"}],\"name\":\"deployToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenAdmin\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"image\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"metadata\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"context\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"originatingChainId\",\"type\":\"uint256\"}],\"internalType\":\"structIClanker.TokenConfig\",\"name\":\"tokenConfig\",\"type\":\"tuple\"}],\"name\":\"deployTokenZeroSupply\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"deploymentInfoForToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"deprecated\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"enabledLockers\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"deprecated_\",\"type\":\"bool\"}],\"name\":\"setDeprecated\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"extension\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setExtension\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setLocker\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"mevModule\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setMevModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"teamFeeRecipient_\",\"type\":\"address\"}],\"name\":\"setTeamFeeRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"teamFeeRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"tokenDeploymentInfo\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"extensions\",\"type\":\"address[]\"}],\"internalType\":\"structIClanker.DeploymentInfo\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ClankerABI is the input ABI used to generate the binding from.
// Deprecated: Use ClankerMetaData.ABI instead.
var ClankerABI = ClankerMetaData.ABI

// Clanker is an auto generated Go binding around an Ethereum contract.
type Clanker struct {
	ClankerCaller     // Read-only binding to the contract
	ClankerTransactor // Write-only binding to the contract
	ClankerFilterer   // Log filterer for contract events
}

// ClankerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClankerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClankerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClankerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClankerSession struct {
	Contract     *Clanker          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClankerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClankerCallerSession struct {
	Contract *ClankerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ClankerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClankerTransactorSession struct {
	Contract     *ClankerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ClankerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClankerRaw struct {
	Contract *Clanker // Generic contract binding to access the raw methods on
}

// ClankerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClankerCallerRaw struct {
	Contract *ClankerCaller // Generic read-only contract binding to access the raw methods on
}

// ClankerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClankerTransactorRaw struct {
	Contract *ClankerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClanker creates a new instance of Clanker, bound to a specific deployed contract.
func NewClanker(address common.Address, backend bind.ContractBackend) (*Clanker, error) {
	contract, err := bindClanker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Clanker{ClankerCaller: ClankerCaller{contract: contract}, ClankerTransactor: ClankerTransactor{contract: contract}, ClankerFilterer: ClankerFilterer{contract: contract}}, nil
}

// NewClankerCaller creates a new read-only instance of Clanker, bound to a specific deployed contract.
func NewClankerCaller(address common.Address, caller bind.ContractCaller) (*ClankerCaller, error) {
	contract, err := bindClanker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClankerCaller{contract: contract}, nil
}

// NewClankerTransactor creates a new write-only instance of Clanker, bound to a specific deployed contract.
func NewClankerTransactor(address common.Address, transactor bind.ContractTransactor) (*ClankerTransactor, error) {
	contract, err := bindClanker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClankerTransactor{contract: contract}, nil
}

// NewClankerFilterer creates a new log filterer instance of Clanker, bound to a specific deployed contract.
func NewClankerFilterer(address common.Address, filterer bind.ContractFilterer) (*ClankerFilterer, error) {
	contract, err := bindClanker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClankerFilterer{contract: contract}, nil
}

// bindClanker binds a generic wrapper to an already deployed contract.
func bindClanker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClankerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Clanker *ClankerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Clanker.Contract.ClankerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Clanker *ClankerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.Contract.ClankerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Clanker *ClankerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Clanker.Contract.ClankerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Clanker *ClankerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Clanker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Clanker *ClankerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Clanker *ClankerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Clanker.Contract.contract.Transact(opts, method, params...)
}

// BPS is a free data retrieval call binding the contract method 0x249d39e9.
//
// Solidity: function BPS() view returns(uint256)
func (_Clanker *ClankerCaller) BPS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "BPS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BPS is a free data retrieval call binding the contract method 0x249d39e9.
//
// Solidity: function BPS() view returns(uint256)
func (_Clanker *ClankerSession) BPS() (*big.Int, error) {
	return _Clanker.Contract.BPS(&_Clanker.CallOpts)
}

// BPS is a free data retrieval call binding the contract method 0x249d39e9.
//
// Solidity: function BPS() view returns(uint256)
func (_Clanker *ClankerCallerSession) BPS() (*big.Int, error) {
	return _Clanker.Contract.BPS(&_Clanker.CallOpts)
}

// MAXEXTENSIONS is a free data retrieval call binding the contract method 0xc07780f6.
//
// Solidity: function MAX_EXTENSIONS() view returns(uint256)
func (_Clanker *ClankerCaller) MAXEXTENSIONS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MAX_EXTENSIONS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXEXTENSIONS is a free data retrieval call binding the contract method 0xc07780f6.
//
// Solidity: function MAX_EXTENSIONS() view returns(uint256)
func (_Clanker *ClankerSession) MAXEXTENSIONS() (*big.Int, error) {
	return _Clanker.Contract.MAXEXTENSIONS(&_Clanker.CallOpts)
}

// MAXEXTENSIONS is a free data retrieval call binding the contract method 0xc07780f6.
//
// Solidity: function MAX_EXTENSIONS() view returns(uint256)
func (_Clanker *ClankerCallerSession) MAXEXTENSIONS() (*big.Int, error) {
	return _Clanker.Contract.MAXEXTENSIONS(&_Clanker.CallOpts)
}

// MAXEXTENSIONBPS is a free data retrieval call binding the contract method 0xcca91a70.
//
// Solidity: function MAX_EXTENSION_BPS() view returns(uint16)
func (_Clanker *ClankerCaller) MAXEXTENSIONBPS(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MAX_EXTENSION_BPS")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MAXEXTENSIONBPS is a free data retrieval call binding the contract method 0xcca91a70.
//
// Solidity: function MAX_EXTENSION_BPS() view returns(uint16)
func (_Clanker *ClankerSession) MAXEXTENSIONBPS() (uint16, error) {
	return _Clanker.Contract.MAXEXTENSIONBPS(&_Clanker.CallOpts)
}

// MAXEXTENSIONBPS is a free data retrieval call binding the contract method 0xcca91a70.
//
// Solidity: function MAX_EXTENSION_BPS() view returns(uint16)
func (_Clanker *ClankerCallerSession) MAXEXTENSIONBPS() (uint16, error) {
	return _Clanker.Contract.MAXEXTENSIONBPS(&_Clanker.CallOpts)
}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_Clanker *ClankerCaller) TOKENSUPPLY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "TOKEN_SUPPLY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_Clanker *ClankerSession) TOKENSUPPLY() (*big.Int, error) {
	return _Clanker.Contract.TOKENSUPPLY(&_Clanker.CallOpts)
}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_Clanker *ClankerCallerSession) TOKENSUPPLY() (*big.Int, error) {
	return _Clanker.Contract.TOKENSUPPLY(&_Clanker.CallOpts)
}

// Admins is a free data retrieval call binding the contract method 0x429b62e5.
//
// Solidity: function admins(address ) view returns(bool)
func (_Clanker *ClankerCaller) Admins(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "admins", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Admins is a free data retrieval call binding the contract method 0x429b62e5.
//
// Solidity: function admins(address ) view returns(bool)
func (_Clanker *ClankerSession) Admins(arg0 common.Address) (bool, error) {
	return _Clanker.Contract.Admins(&_Clanker.CallOpts, arg0)
}

// Admins is a free data retrieval call binding the contract method 0x429b62e5.
//
// Solidity: function admins(address ) view returns(bool)
func (_Clanker *ClankerCallerSession) Admins(arg0 common.Address) (bool, error) {
	return _Clanker.Contract.Admins(&_Clanker.CallOpts, arg0)
}

// DeploymentInfoForToken is a free data retrieval call binding the contract method 0x06562980.
//
// Solidity: function deploymentInfoForToken(address token) view returns(address token, address hook, address locker)
func (_Clanker *ClankerCaller) DeploymentInfoForToken(opts *bind.CallOpts, token common.Address) (struct {
	Token  common.Address
	Hook   common.Address
	Locker common.Address
}, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "deploymentInfoForToken", token)

	outstruct := new(struct {
		Token  common.Address
		Hook   common.Address
		Locker common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Token = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Hook = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Locker = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// DeploymentInfoForToken is a free data retrieval call binding the contract method 0x06562980.
//
// Solidity: function deploymentInfoForToken(address token) view returns(address token, address hook, address locker)
func (_Clanker *ClankerSession) DeploymentInfoForToken(token common.Address) (struct {
	Token  common.Address
	Hook   common.Address
	Locker common.Address
}, error) {
	return _Clanker.Contract.DeploymentInfoForToken(&_Clanker.CallOpts, token)
}

// DeploymentInfoForToken is a free data retrieval call binding the contract method 0x06562980.
//
// Solidity: function deploymentInfoForToken(address token) view returns(address token, address hook, address locker)
func (_Clanker *ClankerCallerSession) DeploymentInfoForToken(token common.Address) (struct {
	Token  common.Address
	Hook   common.Address
	Locker common.Address
}, error) {
	return _Clanker.Contract.DeploymentInfoForToken(&_Clanker.CallOpts, token)
}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_Clanker *ClankerCaller) Deprecated(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "deprecated")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_Clanker *ClankerSession) Deprecated() (bool, error) {
	return _Clanker.Contract.Deprecated(&_Clanker.CallOpts)
}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_Clanker *ClankerCallerSession) Deprecated() (bool, error) {
	return _Clanker.Contract.Deprecated(&_Clanker.CallOpts)
}

// EnabledLockers is a free data retrieval call binding the contract method 0x3c909e39.
//
// Solidity: function enabledLockers(address locker, address hook) view returns(bool enabled)
func (_Clanker *ClankerCaller) EnabledLockers(opts *bind.CallOpts, locker common.Address, hook common.Address) (bool, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "enabledLockers", locker, hook)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// EnabledLockers is a free data retrieval call binding the contract method 0x3c909e39.
//
// Solidity: function enabledLockers(address locker, address hook) view returns(bool enabled)
func (_Clanker *ClankerSession) EnabledLockers(locker common.Address, hook common.Address) (bool, error) {
	return _Clanker.Contract.EnabledLockers(&_Clanker.CallOpts, locker, hook)
}

// EnabledLockers is a free data retrieval call binding the contract method 0x3c909e39.
//
// Solidity: function enabledLockers(address locker, address hook) view returns(bool enabled)
func (_Clanker *ClankerCallerSession) EnabledLockers(locker common.Address, hook common.Address) (bool, error) {
	return _Clanker.Contract.EnabledLockers(&_Clanker.CallOpts, locker, hook)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerSession) Owner() (common.Address, error) {
	return _Clanker.Contract.Owner(&_Clanker.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerCallerSession) Owner() (common.Address, error) {
	return _Clanker.Contract.Owner(&_Clanker.CallOpts)
}

// TeamFeeRecipient is a free data retrieval call binding the contract method 0x228871c5.
//
// Solidity: function teamFeeRecipient() view returns(address)
func (_Clanker *ClankerCaller) TeamFeeRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "teamFeeRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TeamFeeRecipient is a free data retrieval call binding the contract method 0x228871c5.
//
// Solidity: function teamFeeRecipient() view returns(address)
func (_Clanker *ClankerSession) TeamFeeRecipient() (common.Address, error) {
	return _Clanker.Contract.TeamFeeRecipient(&_Clanker.CallOpts)
}

// TeamFeeRecipient is a free data retrieval call binding the contract method 0x228871c5.
//
// Solidity: function teamFeeRecipient() view returns(address)
func (_Clanker *ClankerCallerSession) TeamFeeRecipient() (common.Address, error) {
	return _Clanker.Contract.TeamFeeRecipient(&_Clanker.CallOpts)
}

// TokenDeploymentInfo is a free data retrieval call binding the contract method 0x22adcdb1.
//
// Solidity: function tokenDeploymentInfo(address token) view returns((address,address,address,address[]))
func (_Clanker *ClankerCaller) TokenDeploymentInfo(opts *bind.CallOpts, token common.Address) (IClankerDeploymentInfo, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "tokenDeploymentInfo", token)

	if err != nil {
		return *new(IClankerDeploymentInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(IClankerDeploymentInfo)).(*IClankerDeploymentInfo)

	return out0, err

}

// TokenDeploymentInfo is a free data retrieval call binding the contract method 0x22adcdb1.
//
// Solidity: function tokenDeploymentInfo(address token) view returns((address,address,address,address[]))
func (_Clanker *ClankerSession) TokenDeploymentInfo(token common.Address) (IClankerDeploymentInfo, error) {
	return _Clanker.Contract.TokenDeploymentInfo(&_Clanker.CallOpts, token)
}

// TokenDeploymentInfo is a free data retrieval call binding the contract method 0x22adcdb1.
//
// Solidity: function tokenDeploymentInfo(address token) view returns((address,address,address,address[]))
func (_Clanker *ClankerCallerSession) TokenDeploymentInfo(token common.Address) (IClankerDeploymentInfo, error) {
	return _Clanker.Contract.TokenDeploymentInfo(&_Clanker.CallOpts, token)
}

// ClaimTeamFees is a paid mutator transaction binding the contract method 0x0f0f3a2f.
//
// Solidity: function claimTeamFees(address token) returns()
func (_Clanker *ClankerTransactor) ClaimTeamFees(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "claimTeamFees", token)
}

// ClaimTeamFees is a paid mutator transaction binding the contract method 0x0f0f3a2f.
//
// Solidity: function claimTeamFees(address token) returns()
func (_Clanker *ClankerSession) ClaimTeamFees(token common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.ClaimTeamFees(&_Clanker.TransactOpts, token)
}

// ClaimTeamFees is a paid mutator transaction binding the contract method 0x0f0f3a2f.
//
// Solidity: function claimTeamFees(address token) returns()
func (_Clanker *ClankerTransactorSession) ClaimTeamFees(token common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.ClaimTeamFees(&_Clanker.TransactOpts, token)
}

// DeployToken is a paid mutator transaction binding the contract method 0xdf40224a.
//
// Solidity: function deployToken(((address,string,string,bytes32,string,string,string,uint256),(address,address,int24,int24,bytes),(address,address[],address[],uint16[],int24[],int24[],uint16[],bytes),(address,bytes),(address,uint256,uint16,bytes)[]) deploymentConfig) payable returns(address tokenAddress)
func (_Clanker *ClankerTransactor) DeployToken(opts *bind.TransactOpts, deploymentConfig IClankerDeploymentConfig) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "deployToken", deploymentConfig)
}

// DeployToken is a paid mutator transaction binding the contract method 0xdf40224a.
//
// Solidity: function deployToken(((address,string,string,bytes32,string,string,string,uint256),(address,address,int24,int24,bytes),(address,address[],address[],uint16[],int24[],int24[],uint16[],bytes),(address,bytes),(address,uint256,uint16,bytes)[]) deploymentConfig) payable returns(address tokenAddress)
func (_Clanker *ClankerSession) DeployToken(deploymentConfig IClankerDeploymentConfig) (*types.Transaction, error) {
	return _Clanker.Contract.DeployToken(&_Clanker.TransactOpts, deploymentConfig)
}

// DeployToken is a paid mutator transaction binding the contract method 0xdf40224a.
//
// Solidity: function deployToken(((address,string,string,bytes32,string,string,string,uint256),(address,address,int24,int24,bytes),(address,address[],address[],uint16[],int24[],int24[],uint16[],bytes),(address,bytes),(address,uint256,uint16,bytes)[]) deploymentConfig) payable returns(address tokenAddress)
func (_Clanker *ClankerTransactorSession) DeployToken(deploymentConfig IClankerDeploymentConfig) (*types.Transaction, error) {
	return _Clanker.Contract.DeployToken(&_Clanker.TransactOpts, deploymentConfig)
}

// DeployTokenZeroSupply is a paid mutator transaction binding the contract method 0xa238f07f.
//
// Solidity: function deployTokenZeroSupply((address,string,string,bytes32,string,string,string,uint256) tokenConfig) returns(address tokenAddress)
func (_Clanker *ClankerTransactor) DeployTokenZeroSupply(opts *bind.TransactOpts, tokenConfig IClankerTokenConfig) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "deployTokenZeroSupply", tokenConfig)
}

// DeployTokenZeroSupply is a paid mutator transaction binding the contract method 0xa238f07f.
//
// Solidity: function deployTokenZeroSupply((address,string,string,bytes32,string,string,string,uint256) tokenConfig) returns(address tokenAddress)
func (_Clanker *ClankerSession) DeployTokenZeroSupply(tokenConfig IClankerTokenConfig) (*types.Transaction, error) {
	return _Clanker.Contract.DeployTokenZeroSupply(&_Clanker.TransactOpts, tokenConfig)
}

// DeployTokenZeroSupply is a paid mutator transaction binding the contract method 0xa238f07f.
//
// Solidity: function deployTokenZeroSupply((address,string,string,bytes32,string,string,string,uint256) tokenConfig) returns(address tokenAddress)
func (_Clanker *ClankerTransactorSession) DeployTokenZeroSupply(tokenConfig IClankerTokenConfig) (*types.Transaction, error) {
	return _Clanker.Contract.DeployTokenZeroSupply(&_Clanker.TransactOpts, tokenConfig)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerSession) RenounceOwnership() (*types.Transaction, error) {
	return _Clanker.Contract.RenounceOwnership(&_Clanker.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Clanker.Contract.RenounceOwnership(&_Clanker.TransactOpts)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_Clanker *ClankerTransactor) SetAdmin(opts *bind.TransactOpts, admin common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setAdmin", admin, enabled)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_Clanker *ClankerSession) SetAdmin(admin common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetAdmin(&_Clanker.TransactOpts, admin, enabled)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_Clanker *ClankerTransactorSession) SetAdmin(admin common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetAdmin(&_Clanker.TransactOpts, admin, enabled)
}

// SetDeprecated is a paid mutator transaction binding the contract method 0xd848dee7.
//
// Solidity: function setDeprecated(bool deprecated_) returns()
func (_Clanker *ClankerTransactor) SetDeprecated(opts *bind.TransactOpts, deprecated_ bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setDeprecated", deprecated_)
}

// SetDeprecated is a paid mutator transaction binding the contract method 0xd848dee7.
//
// Solidity: function setDeprecated(bool deprecated_) returns()
func (_Clanker *ClankerSession) SetDeprecated(deprecated_ bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetDeprecated(&_Clanker.TransactOpts, deprecated_)
}

// SetDeprecated is a paid mutator transaction binding the contract method 0xd848dee7.
//
// Solidity: function setDeprecated(bool deprecated_) returns()
func (_Clanker *ClankerTransactorSession) SetDeprecated(deprecated_ bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetDeprecated(&_Clanker.TransactOpts, deprecated_)
}

// SetExtension is a paid mutator transaction binding the contract method 0x031bc1ba.
//
// Solidity: function setExtension(address extension, bool enabled) returns()
func (_Clanker *ClankerTransactor) SetExtension(opts *bind.TransactOpts, extension common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setExtension", extension, enabled)
}

// SetExtension is a paid mutator transaction binding the contract method 0x031bc1ba.
//
// Solidity: function setExtension(address extension, bool enabled) returns()
func (_Clanker *ClankerSession) SetExtension(extension common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetExtension(&_Clanker.TransactOpts, extension, enabled)
}

// SetExtension is a paid mutator transaction binding the contract method 0x031bc1ba.
//
// Solidity: function setExtension(address extension, bool enabled) returns()
func (_Clanker *ClankerTransactorSession) SetExtension(extension common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetExtension(&_Clanker.TransactOpts, extension, enabled)
}

// SetHook is a paid mutator transaction binding the contract method 0x833e8db1.
//
// Solidity: function setHook(address hook, bool enabled) returns()
func (_Clanker *ClankerTransactor) SetHook(opts *bind.TransactOpts, hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setHook", hook, enabled)
}

// SetHook is a paid mutator transaction binding the contract method 0x833e8db1.
//
// Solidity: function setHook(address hook, bool enabled) returns()
func (_Clanker *ClankerSession) SetHook(hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetHook(&_Clanker.TransactOpts, hook, enabled)
}

// SetHook is a paid mutator transaction binding the contract method 0x833e8db1.
//
// Solidity: function setHook(address hook, bool enabled) returns()
func (_Clanker *ClankerTransactorSession) SetHook(hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetHook(&_Clanker.TransactOpts, hook, enabled)
}

// SetLocker is a paid mutator transaction binding the contract method 0xfc1b6022.
//
// Solidity: function setLocker(address locker, address hook, bool enabled) returns()
func (_Clanker *ClankerTransactor) SetLocker(opts *bind.TransactOpts, locker common.Address, hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setLocker", locker, hook, enabled)
}

// SetLocker is a paid mutator transaction binding the contract method 0xfc1b6022.
//
// Solidity: function setLocker(address locker, address hook, bool enabled) returns()
func (_Clanker *ClankerSession) SetLocker(locker common.Address, hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetLocker(&_Clanker.TransactOpts, locker, hook, enabled)
}

// SetLocker is a paid mutator transaction binding the contract method 0xfc1b6022.
//
// Solidity: function setLocker(address locker, address hook, bool enabled) returns()
func (_Clanker *ClankerTransactorSession) SetLocker(locker common.Address, hook common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetLocker(&_Clanker.TransactOpts, locker, hook, enabled)
}

// SetMevModule is a paid mutator transaction binding the contract method 0xdbb0b1c7.
//
// Solidity: function setMevModule(address mevModule, bool enabled) returns()
func (_Clanker *ClankerTransactor) SetMevModule(opts *bind.TransactOpts, mevModule common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setMevModule", mevModule, enabled)
}

// SetMevModule is a paid mutator transaction binding the contract method 0xdbb0b1c7.
//
// Solidity: function setMevModule(address mevModule, bool enabled) returns()
func (_Clanker *ClankerSession) SetMevModule(mevModule common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetMevModule(&_Clanker.TransactOpts, mevModule, enabled)
}

// SetMevModule is a paid mutator transaction binding the contract method 0xdbb0b1c7.
//
// Solidity: function setMevModule(address mevModule, bool enabled) returns()
func (_Clanker *ClankerTransactorSession) SetMevModule(mevModule common.Address, enabled bool) (*types.Transaction, error) {
	return _Clanker.Contract.SetMevModule(&_Clanker.TransactOpts, mevModule, enabled)
}

// SetTeamFeeRecipient is a paid mutator transaction binding the contract method 0x926ff255.
//
// Solidity: function setTeamFeeRecipient(address teamFeeRecipient_) returns()
func (_Clanker *ClankerTransactor) SetTeamFeeRecipient(opts *bind.TransactOpts, teamFeeRecipient_ common.Address) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "setTeamFeeRecipient", teamFeeRecipient_)
}

// SetTeamFeeRecipient is a paid mutator transaction binding the contract method 0x926ff255.
//
// Solidity: function setTeamFeeRecipient(address teamFeeRecipient_) returns()
func (_Clanker *ClankerSession) SetTeamFeeRecipient(teamFeeRecipient_ common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.SetTeamFeeRecipient(&_Clanker.TransactOpts, teamFeeRecipient_)
}

// SetTeamFeeRecipient is a paid mutator transaction binding the contract method 0x926ff255.
//
// Solidity: function setTeamFeeRecipient(address teamFeeRecipient_) returns()
func (_Clanker *ClankerTransactorSession) SetTeamFeeRecipient(teamFeeRecipient_ common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.SetTeamFeeRecipient(&_Clanker.TransactOpts, teamFeeRecipient_)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.TransferOwnership(&_Clanker.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.TransferOwnership(&_Clanker.TransactOpts, newOwner)
}

// ClankerClaimTeamFeesIterator is returned from FilterClaimTeamFees and is used to iterate over the raw logs and unpacked data for ClaimTeamFees events raised by the Clanker contract.
type ClankerClaimTeamFeesIterator struct {
	Event *ClankerClaimTeamFees // Event containing the contract specifics and raw log

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
func (it *ClankerClaimTeamFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerClaimTeamFees)
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
		it.Event = new(ClankerClaimTeamFees)
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
func (it *ClankerClaimTeamFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerClaimTeamFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerClaimTeamFees represents a ClaimTeamFees event raised by the Clanker contract.
type ClankerClaimTeamFees struct {
	Token     common.Address
	Recipient common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterClaimTeamFees is a free log retrieval operation binding the contract event 0xe5b9f4d1f6cf7c3238f0c0b48597a9ccd3ff3d2309f3ccd30bd46aad5e06a638.
//
// Solidity: event ClaimTeamFees(address indexed token, address indexed recipient, uint256 amount)
func (_Clanker *ClankerFilterer) FilterClaimTeamFees(opts *bind.FilterOpts, token []common.Address, recipient []common.Address) (*ClankerClaimTeamFeesIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "ClaimTeamFees", tokenRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &ClankerClaimTeamFeesIterator{contract: _Clanker.contract, event: "ClaimTeamFees", logs: logs, sub: sub}, nil
}

// WatchClaimTeamFees is a free log subscription operation binding the contract event 0xe5b9f4d1f6cf7c3238f0c0b48597a9ccd3ff3d2309f3ccd30bd46aad5e06a638.
//
// Solidity: event ClaimTeamFees(address indexed token, address indexed recipient, uint256 amount)
func (_Clanker *ClankerFilterer) WatchClaimTeamFees(opts *bind.WatchOpts, sink chan<- *ClankerClaimTeamFees, token []common.Address, recipient []common.Address) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "ClaimTeamFees", tokenRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerClaimTeamFees)
				if err := _Clanker.contract.UnpackLog(event, "ClaimTeamFees", log); err != nil {
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

// ParseClaimTeamFees is a log parse operation binding the contract event 0xe5b9f4d1f6cf7c3238f0c0b48597a9ccd3ff3d2309f3ccd30bd46aad5e06a638.
//
// Solidity: event ClaimTeamFees(address indexed token, address indexed recipient, uint256 amount)
func (_Clanker *ClankerFilterer) ParseClaimTeamFees(log types.Log) (*ClankerClaimTeamFees, error) {
	event := new(ClankerClaimTeamFees)
	if err := _Clanker.contract.UnpackLog(event, "ClaimTeamFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerExtensionTriggeredIterator is returned from FilterExtensionTriggered and is used to iterate over the raw logs and unpacked data for ExtensionTriggered events raised by the Clanker contract.
type ClankerExtensionTriggeredIterator struct {
	Event *ClankerExtensionTriggered // Event containing the contract specifics and raw log

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
func (it *ClankerExtensionTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerExtensionTriggered)
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
		it.Event = new(ClankerExtensionTriggered)
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
func (it *ClankerExtensionTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerExtensionTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerExtensionTriggered represents a ExtensionTriggered event raised by the Clanker contract.
type ClankerExtensionTriggered struct {
	Extension       common.Address
	ExtensionSupply *big.Int
	MsgValue        *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterExtensionTriggered is a free log retrieval operation binding the contract event 0xe80ed94c33183ba307727bf230f18d40178975f51b301a8415b90f4c9f549b7f.
//
// Solidity: event ExtensionTriggered(address extension, uint256 extensionSupply, uint256 msgValue)
func (_Clanker *ClankerFilterer) FilterExtensionTriggered(opts *bind.FilterOpts) (*ClankerExtensionTriggeredIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "ExtensionTriggered")
	if err != nil {
		return nil, err
	}
	return &ClankerExtensionTriggeredIterator{contract: _Clanker.contract, event: "ExtensionTriggered", logs: logs, sub: sub}, nil
}

// WatchExtensionTriggered is a free log subscription operation binding the contract event 0xe80ed94c33183ba307727bf230f18d40178975f51b301a8415b90f4c9f549b7f.
//
// Solidity: event ExtensionTriggered(address extension, uint256 extensionSupply, uint256 msgValue)
func (_Clanker *ClankerFilterer) WatchExtensionTriggered(opts *bind.WatchOpts, sink chan<- *ClankerExtensionTriggered) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "ExtensionTriggered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerExtensionTriggered)
				if err := _Clanker.contract.UnpackLog(event, "ExtensionTriggered", log); err != nil {
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

// ParseExtensionTriggered is a log parse operation binding the contract event 0xe80ed94c33183ba307727bf230f18d40178975f51b301a8415b90f4c9f549b7f.
//
// Solidity: event ExtensionTriggered(address extension, uint256 extensionSupply, uint256 msgValue)
func (_Clanker *ClankerFilterer) ParseExtensionTriggered(log types.Log) (*ClankerExtensionTriggered, error) {
	event := new(ClankerExtensionTriggered)
	if err := _Clanker.contract.UnpackLog(event, "ExtensionTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Clanker contract.
type ClankerOwnershipTransferredIterator struct {
	Event *ClankerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ClankerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerOwnershipTransferred)
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
		it.Event = new(ClankerOwnershipTransferred)
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
func (it *ClankerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerOwnershipTransferred represents a OwnershipTransferred event raised by the Clanker contract.
type ClankerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Clanker *ClankerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ClankerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ClankerOwnershipTransferredIterator{contract: _Clanker.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Clanker *ClankerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ClankerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerOwnershipTransferred)
				if err := _Clanker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Clanker *ClankerFilterer) ParseOwnershipTransferred(log types.Log) (*ClankerOwnershipTransferred, error) {
	event := new(ClankerOwnershipTransferred)
	if err := _Clanker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetAdminIterator is returned from FilterSetAdmin and is used to iterate over the raw logs and unpacked data for SetAdmin events raised by the Clanker contract.
type ClankerSetAdminIterator struct {
	Event *ClankerSetAdmin // Event containing the contract specifics and raw log

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
func (it *ClankerSetAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetAdmin)
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
		it.Event = new(ClankerSetAdmin)
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
func (it *ClankerSetAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetAdmin represents a SetAdmin event raised by the Clanker contract.
type ClankerSetAdmin struct {
	Admin   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSetAdmin is a free log retrieval operation binding the contract event 0x55a5194bc0174fcaf12b2978bef43911466bf63b34db8d1dd1a0d5dcd5c41bea.
//
// Solidity: event SetAdmin(address indexed admin, bool enabled)
func (_Clanker *ClankerFilterer) FilterSetAdmin(opts *bind.FilterOpts, admin []common.Address) (*ClankerSetAdminIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return &ClankerSetAdminIterator{contract: _Clanker.contract, event: "SetAdmin", logs: logs, sub: sub}, nil
}

// WatchSetAdmin is a free log subscription operation binding the contract event 0x55a5194bc0174fcaf12b2978bef43911466bf63b34db8d1dd1a0d5dcd5c41bea.
//
// Solidity: event SetAdmin(address indexed admin, bool enabled)
func (_Clanker *ClankerFilterer) WatchSetAdmin(opts *bind.WatchOpts, sink chan<- *ClankerSetAdmin, admin []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetAdmin)
				if err := _Clanker.contract.UnpackLog(event, "SetAdmin", log); err != nil {
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

// ParseSetAdmin is a log parse operation binding the contract event 0x55a5194bc0174fcaf12b2978bef43911466bf63b34db8d1dd1a0d5dcd5c41bea.
//
// Solidity: event SetAdmin(address indexed admin, bool enabled)
func (_Clanker *ClankerFilterer) ParseSetAdmin(log types.Log) (*ClankerSetAdmin, error) {
	event := new(ClankerSetAdmin)
	if err := _Clanker.contract.UnpackLog(event, "SetAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetDeprecatedIterator is returned from FilterSetDeprecated and is used to iterate over the raw logs and unpacked data for SetDeprecated events raised by the Clanker contract.
type ClankerSetDeprecatedIterator struct {
	Event *ClankerSetDeprecated // Event containing the contract specifics and raw log

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
func (it *ClankerSetDeprecatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetDeprecated)
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
		it.Event = new(ClankerSetDeprecated)
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
func (it *ClankerSetDeprecatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetDeprecatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetDeprecated represents a SetDeprecated event raised by the Clanker contract.
type ClankerSetDeprecated struct {
	Deprecated bool
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSetDeprecated is a free log retrieval operation binding the contract event 0x20db9067ad1976f7d6ee4ee07eea48c1139e0716fd856ec6edf203236c37db82.
//
// Solidity: event SetDeprecated(bool deprecated)
func (_Clanker *ClankerFilterer) FilterSetDeprecated(opts *bind.FilterOpts) (*ClankerSetDeprecatedIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetDeprecated")
	if err != nil {
		return nil, err
	}
	return &ClankerSetDeprecatedIterator{contract: _Clanker.contract, event: "SetDeprecated", logs: logs, sub: sub}, nil
}

// WatchSetDeprecated is a free log subscription operation binding the contract event 0x20db9067ad1976f7d6ee4ee07eea48c1139e0716fd856ec6edf203236c37db82.
//
// Solidity: event SetDeprecated(bool deprecated)
func (_Clanker *ClankerFilterer) WatchSetDeprecated(opts *bind.WatchOpts, sink chan<- *ClankerSetDeprecated) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetDeprecated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetDeprecated)
				if err := _Clanker.contract.UnpackLog(event, "SetDeprecated", log); err != nil {
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

// ParseSetDeprecated is a log parse operation binding the contract event 0x20db9067ad1976f7d6ee4ee07eea48c1139e0716fd856ec6edf203236c37db82.
//
// Solidity: event SetDeprecated(bool deprecated)
func (_Clanker *ClankerFilterer) ParseSetDeprecated(log types.Log) (*ClankerSetDeprecated, error) {
	event := new(ClankerSetDeprecated)
	if err := _Clanker.contract.UnpackLog(event, "SetDeprecated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetExtensionIterator is returned from FilterSetExtension and is used to iterate over the raw logs and unpacked data for SetExtension events raised by the Clanker contract.
type ClankerSetExtensionIterator struct {
	Event *ClankerSetExtension // Event containing the contract specifics and raw log

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
func (it *ClankerSetExtensionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetExtension)
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
		it.Event = new(ClankerSetExtension)
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
func (it *ClankerSetExtensionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetExtensionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetExtension represents a SetExtension event raised by the Clanker contract.
type ClankerSetExtension struct {
	Extension common.Address
	Enabled   bool
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSetExtension is a free log retrieval operation binding the contract event 0xb393434b606480a39b9a3b6f498a7f5312283d75b93f92d1dffc0dae86425ea2.
//
// Solidity: event SetExtension(address extension, bool enabled)
func (_Clanker *ClankerFilterer) FilterSetExtension(opts *bind.FilterOpts) (*ClankerSetExtensionIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetExtension")
	if err != nil {
		return nil, err
	}
	return &ClankerSetExtensionIterator{contract: _Clanker.contract, event: "SetExtension", logs: logs, sub: sub}, nil
}

// WatchSetExtension is a free log subscription operation binding the contract event 0xb393434b606480a39b9a3b6f498a7f5312283d75b93f92d1dffc0dae86425ea2.
//
// Solidity: event SetExtension(address extension, bool enabled)
func (_Clanker *ClankerFilterer) WatchSetExtension(opts *bind.WatchOpts, sink chan<- *ClankerSetExtension) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetExtension")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetExtension)
				if err := _Clanker.contract.UnpackLog(event, "SetExtension", log); err != nil {
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

// ParseSetExtension is a log parse operation binding the contract event 0xb393434b606480a39b9a3b6f498a7f5312283d75b93f92d1dffc0dae86425ea2.
//
// Solidity: event SetExtension(address extension, bool enabled)
func (_Clanker *ClankerFilterer) ParseSetExtension(log types.Log) (*ClankerSetExtension, error) {
	event := new(ClankerSetExtension)
	if err := _Clanker.contract.UnpackLog(event, "SetExtension", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetHookIterator is returned from FilterSetHook and is used to iterate over the raw logs and unpacked data for SetHook events raised by the Clanker contract.
type ClankerSetHookIterator struct {
	Event *ClankerSetHook // Event containing the contract specifics and raw log

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
func (it *ClankerSetHookIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetHook)
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
		it.Event = new(ClankerSetHook)
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
func (it *ClankerSetHookIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetHookIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetHook represents a SetHook event raised by the Clanker contract.
type ClankerSetHook struct {
	Hook    common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSetHook is a free log retrieval operation binding the contract event 0x3e7510e93c4fe81ab57ed70b4eaa9a407eeced9cdb0954a360b36947f38ccda0.
//
// Solidity: event SetHook(address hook, bool enabled)
func (_Clanker *ClankerFilterer) FilterSetHook(opts *bind.FilterOpts) (*ClankerSetHookIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetHook")
	if err != nil {
		return nil, err
	}
	return &ClankerSetHookIterator{contract: _Clanker.contract, event: "SetHook", logs: logs, sub: sub}, nil
}

// WatchSetHook is a free log subscription operation binding the contract event 0x3e7510e93c4fe81ab57ed70b4eaa9a407eeced9cdb0954a360b36947f38ccda0.
//
// Solidity: event SetHook(address hook, bool enabled)
func (_Clanker *ClankerFilterer) WatchSetHook(opts *bind.WatchOpts, sink chan<- *ClankerSetHook) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetHook")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetHook)
				if err := _Clanker.contract.UnpackLog(event, "SetHook", log); err != nil {
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

// ParseSetHook is a log parse operation binding the contract event 0x3e7510e93c4fe81ab57ed70b4eaa9a407eeced9cdb0954a360b36947f38ccda0.
//
// Solidity: event SetHook(address hook, bool enabled)
func (_Clanker *ClankerFilterer) ParseSetHook(log types.Log) (*ClankerSetHook, error) {
	event := new(ClankerSetHook)
	if err := _Clanker.contract.UnpackLog(event, "SetHook", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetLockerIterator is returned from FilterSetLocker and is used to iterate over the raw logs and unpacked data for SetLocker events raised by the Clanker contract.
type ClankerSetLockerIterator struct {
	Event *ClankerSetLocker // Event containing the contract specifics and raw log

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
func (it *ClankerSetLockerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetLocker)
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
		it.Event = new(ClankerSetLocker)
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
func (it *ClankerSetLockerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetLockerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetLocker represents a SetLocker event raised by the Clanker contract.
type ClankerSetLocker struct {
	Locker  common.Address
	Hook    common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSetLocker is a free log retrieval operation binding the contract event 0xa42b595aa29ec637074f31538fbf673587d74ba490265a33cf04ed41d27a9ddc.
//
// Solidity: event SetLocker(address locker, address hook, bool enabled)
func (_Clanker *ClankerFilterer) FilterSetLocker(opts *bind.FilterOpts) (*ClankerSetLockerIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetLocker")
	if err != nil {
		return nil, err
	}
	return &ClankerSetLockerIterator{contract: _Clanker.contract, event: "SetLocker", logs: logs, sub: sub}, nil
}

// WatchSetLocker is a free log subscription operation binding the contract event 0xa42b595aa29ec637074f31538fbf673587d74ba490265a33cf04ed41d27a9ddc.
//
// Solidity: event SetLocker(address locker, address hook, bool enabled)
func (_Clanker *ClankerFilterer) WatchSetLocker(opts *bind.WatchOpts, sink chan<- *ClankerSetLocker) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetLocker")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetLocker)
				if err := _Clanker.contract.UnpackLog(event, "SetLocker", log); err != nil {
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

// ParseSetLocker is a log parse operation binding the contract event 0xa42b595aa29ec637074f31538fbf673587d74ba490265a33cf04ed41d27a9ddc.
//
// Solidity: event SetLocker(address locker, address hook, bool enabled)
func (_Clanker *ClankerFilterer) ParseSetLocker(log types.Log) (*ClankerSetLocker, error) {
	event := new(ClankerSetLocker)
	if err := _Clanker.contract.UnpackLog(event, "SetLocker", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetMevModuleIterator is returned from FilterSetMevModule and is used to iterate over the raw logs and unpacked data for SetMevModule events raised by the Clanker contract.
type ClankerSetMevModuleIterator struct {
	Event *ClankerSetMevModule // Event containing the contract specifics and raw log

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
func (it *ClankerSetMevModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetMevModule)
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
		it.Event = new(ClankerSetMevModule)
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
func (it *ClankerSetMevModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetMevModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetMevModule represents a SetMevModule event raised by the Clanker contract.
type ClankerSetMevModule struct {
	MevModule common.Address
	Enabled   bool
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSetMevModule is a free log retrieval operation binding the contract event 0xc4bf3ad397ab835bf8aefef70e7e1c35a0cfad7be9c060049050303ace20605c.
//
// Solidity: event SetMevModule(address mevModule, bool enabled)
func (_Clanker *ClankerFilterer) FilterSetMevModule(opts *bind.FilterOpts) (*ClankerSetMevModuleIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetMevModule")
	if err != nil {
		return nil, err
	}
	return &ClankerSetMevModuleIterator{contract: _Clanker.contract, event: "SetMevModule", logs: logs, sub: sub}, nil
}

// WatchSetMevModule is a free log subscription operation binding the contract event 0xc4bf3ad397ab835bf8aefef70e7e1c35a0cfad7be9c060049050303ace20605c.
//
// Solidity: event SetMevModule(address mevModule, bool enabled)
func (_Clanker *ClankerFilterer) WatchSetMevModule(opts *bind.WatchOpts, sink chan<- *ClankerSetMevModule) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetMevModule")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetMevModule)
				if err := _Clanker.contract.UnpackLog(event, "SetMevModule", log); err != nil {
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

// ParseSetMevModule is a log parse operation binding the contract event 0xc4bf3ad397ab835bf8aefef70e7e1c35a0cfad7be9c060049050303ace20605c.
//
// Solidity: event SetMevModule(address mevModule, bool enabled)
func (_Clanker *ClankerFilterer) ParseSetMevModule(log types.Log) (*ClankerSetMevModule, error) {
	event := new(ClankerSetMevModule)
	if err := _Clanker.contract.UnpackLog(event, "SetMevModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerSetTeamFeeRecipientIterator is returned from FilterSetTeamFeeRecipient and is used to iterate over the raw logs and unpacked data for SetTeamFeeRecipient events raised by the Clanker contract.
type ClankerSetTeamFeeRecipientIterator struct {
	Event *ClankerSetTeamFeeRecipient // Event containing the contract specifics and raw log

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
func (it *ClankerSetTeamFeeRecipientIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerSetTeamFeeRecipient)
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
		it.Event = new(ClankerSetTeamFeeRecipient)
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
func (it *ClankerSetTeamFeeRecipientIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerSetTeamFeeRecipientIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerSetTeamFeeRecipient represents a SetTeamFeeRecipient event raised by the Clanker contract.
type ClankerSetTeamFeeRecipient struct {
	OldTeamFeeRecipient common.Address
	NewTeamFeeRecipient common.Address
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSetTeamFeeRecipient is a free log retrieval operation binding the contract event 0xb4a70636e7a4c9d224bbe13304010ccb0ac964b3cc46c612d9abaeacfa204bd9.
//
// Solidity: event SetTeamFeeRecipient(address oldTeamFeeRecipient, address newTeamFeeRecipient)
func (_Clanker *ClankerFilterer) FilterSetTeamFeeRecipient(opts *bind.FilterOpts) (*ClankerSetTeamFeeRecipientIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "SetTeamFeeRecipient")
	if err != nil {
		return nil, err
	}
	return &ClankerSetTeamFeeRecipientIterator{contract: _Clanker.contract, event: "SetTeamFeeRecipient", logs: logs, sub: sub}, nil
}

// WatchSetTeamFeeRecipient is a free log subscription operation binding the contract event 0xb4a70636e7a4c9d224bbe13304010ccb0ac964b3cc46c612d9abaeacfa204bd9.
//
// Solidity: event SetTeamFeeRecipient(address oldTeamFeeRecipient, address newTeamFeeRecipient)
func (_Clanker *ClankerFilterer) WatchSetTeamFeeRecipient(opts *bind.WatchOpts, sink chan<- *ClankerSetTeamFeeRecipient) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "SetTeamFeeRecipient")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerSetTeamFeeRecipient)
				if err := _Clanker.contract.UnpackLog(event, "SetTeamFeeRecipient", log); err != nil {
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

// ParseSetTeamFeeRecipient is a log parse operation binding the contract event 0xb4a70636e7a4c9d224bbe13304010ccb0ac964b3cc46c612d9abaeacfa204bd9.
//
// Solidity: event SetTeamFeeRecipient(address oldTeamFeeRecipient, address newTeamFeeRecipient)
func (_Clanker *ClankerFilterer) ParseSetTeamFeeRecipient(log types.Log) (*ClankerSetTeamFeeRecipient, error) {
	event := new(ClankerSetTeamFeeRecipient)
	if err := _Clanker.contract.UnpackLog(event, "SetTeamFeeRecipient", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerTokenCreatedIterator is returned from FilterTokenCreated and is used to iterate over the raw logs and unpacked data for TokenCreated events raised by the Clanker contract.
type ClankerTokenCreatedIterator struct {
	Event *ClankerTokenCreated // Event containing the contract specifics and raw log

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
func (it *ClankerTokenCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerTokenCreated)
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
		it.Event = new(ClankerTokenCreated)
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
func (it *ClankerTokenCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerTokenCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerTokenCreated represents a TokenCreated event raised by the Clanker contract.
type ClankerTokenCreated struct {
	MsgSender        common.Address
	TokenAddress     common.Address
	TokenAdmin       common.Address
	TokenImage       string
	TokenName        string
	TokenSymbol      string
	TokenMetadata    string
	TokenContext     string
	StartingTick     *big.Int
	PoolHook         common.Address
	PoolId           [32]byte
	PairedToken      common.Address
	Locker           common.Address
	MevModule        common.Address
	ExtensionsSupply *big.Int
	Extensions       []common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterTokenCreated is a free log retrieval operation binding the contract event 0x9299d1d1a88d8e1abdc591ae7a167a6bc63a8f17d695804e9091ee33aa89fb67.
//
// Solidity: event TokenCreated(address msgSender, address indexed tokenAddress, address indexed tokenAdmin, string tokenImage, string tokenName, string tokenSymbol, string tokenMetadata, string tokenContext, int24 startingTick, address poolHook, bytes32 poolId, address pairedToken, address locker, address mevModule, uint256 extensionsSupply, address[] extensions)
func (_Clanker *ClankerFilterer) FilterTokenCreated(opts *bind.FilterOpts, tokenAddress []common.Address, tokenAdmin []common.Address) (*ClankerTokenCreatedIterator, error) {

	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}
	var tokenAdminRule []interface{}
	for _, tokenAdminItem := range tokenAdmin {
		tokenAdminRule = append(tokenAdminRule, tokenAdminItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "TokenCreated", tokenAddressRule, tokenAdminRule)
	if err != nil {
		return nil, err
	}
	return &ClankerTokenCreatedIterator{contract: _Clanker.contract, event: "TokenCreated", logs: logs, sub: sub}, nil
}

// WatchTokenCreated is a free log subscription operation binding the contract event 0x9299d1d1a88d8e1abdc591ae7a167a6bc63a8f17d695804e9091ee33aa89fb67.
//
// Solidity: event TokenCreated(address msgSender, address indexed tokenAddress, address indexed tokenAdmin, string tokenImage, string tokenName, string tokenSymbol, string tokenMetadata, string tokenContext, int24 startingTick, address poolHook, bytes32 poolId, address pairedToken, address locker, address mevModule, uint256 extensionsSupply, address[] extensions)
func (_Clanker *ClankerFilterer) WatchTokenCreated(opts *bind.WatchOpts, sink chan<- *ClankerTokenCreated, tokenAddress []common.Address, tokenAdmin []common.Address) (event.Subscription, error) {

	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}
	var tokenAdminRule []interface{}
	for _, tokenAdminItem := range tokenAdmin {
		tokenAdminRule = append(tokenAdminRule, tokenAdminItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "TokenCreated", tokenAddressRule, tokenAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerTokenCreated)
				if err := _Clanker.contract.UnpackLog(event, "TokenCreated", log); err != nil {
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

// ParseTokenCreated is a log parse operation binding the contract event 0x9299d1d1a88d8e1abdc591ae7a167a6bc63a8f17d695804e9091ee33aa89fb67.
//
// Solidity: event TokenCreated(address msgSender, address indexed tokenAddress, address indexed tokenAdmin, string tokenImage, string tokenName, string tokenSymbol, string tokenMetadata, string tokenContext, int24 startingTick, address poolHook, bytes32 poolId, address pairedToken, address locker, address mevModule, uint256 extensionsSupply, address[] extensions)
func (_Clanker *ClankerFilterer) ParseTokenCreated(log types.Log) (*ClankerTokenCreated, error) {
	event := new(ClankerTokenCreated)
	if err := _Clanker.contract.UnpackLog(event, "TokenCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
