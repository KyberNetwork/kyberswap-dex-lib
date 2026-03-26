package testutil

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-resty/resty/v2"

	gojson "github.com/goccy/go-json"
)

// Tenderly environment variable names.
const (
	envTenderlyToken     = "TENDERLY_ACCESS_KEY"
	envTenderlyAccountID = "TENDERLY_ACCOUNT_ID"
	envTenderlyProject   = "TENDERLY_PROJECT"
)

// Default addresses for state overrides.
var (
	DefaultSender  = common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	DefaultSpender = common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	defaultSenderLower = strings.ToLower(DefaultSender.Hex())

	maxUint256      = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	defaultGasPrice = big.NewInt(2_000_000_000)
)

// --- Types ---

// StateObject represents storage overrides for a single contract.
type StateObject struct {
	Storage map[string]string `json:"storage,omitempty"`
	Balance string            `json:"balance,omitempty"`
}

// StateOverride maps contract addresses to their storage overrides.
type StateOverride map[string]*StateObject

// SimulateRequest is the request payload for Tenderly simulation API.
type SimulateRequest struct {
	NetworkID      string        `json:"network_id"`
	From           string        `json:"from"`
	To             string        `json:"to"`
	Input          string        `json:"input"`
	Gas            int64         `json:"gas,omitempty"`
	GasPrice       string        `json:"gas_price,omitempty"`
	Value          string        `json:"value,omitempty"`
	Save           bool          `json:"save"`
	SaveIfFails    bool          `json:"save_if_fails"`
	SimulationType string        `json:"simulation_type,omitempty"` // "full" (default), "quick", "abi"
	StateObjects   StateOverride `json:"state_objects,omitempty"`
	BlockNumber    uint64        `json:"block_number,omitempty"`
}

// SimulationResult contains the simulation response from Tenderly.
type SimulationResult struct {
	Transaction struct {
		Hash            string `json:"hash"`
		Status          bool   `json:"status"`
		GasUsed         int64  `json:"gas_used"`
		TransactionInfo struct {
			CallTrace callTrace `json:"call_trace"`
		} `json:"transaction_info"`
	} `json:"transaction"`
	Simulation struct {
		ID      string `json:"id"`
		Status  bool   `json:"status"`
		GasUsed int64  `json:"gas_used"`
	} `json:"simulation"`
}

// GasEstimateResult holds the comparison between simulated and estimated gas.
type GasEstimateResult struct {
	SimulatedGas int64
	EstimatedGas int64
	Difference   int64
	DiffPercent  float64
	SimulationID string
	TxStatus     bool
}

// --- Client ---

// TenderlyClient wraps the Tenderly Simulation API for gas estimation.
type TenderlyClient struct {
	client    *resty.Client
	accountID string
	project   string
}

// NewTenderlyClient creates a client from environment variables.
// Returns nil if env vars are not set.
func NewTenderlyClient() *TenderlyClient {
	token := os.Getenv(envTenderlyToken)
	accountID := os.Getenv(envTenderlyAccountID)
	project := os.Getenv(envTenderlyProject)

	token = "f2Xsk34pz27gNN4uZ019NcDrXsL6gDJT"
	accountID = "tenderly-kyber"
	project = "nhathm"

	if token == "" || accountID == "" || project == "" {
		return nil
	}

	client := resty.New().
		SetBaseURL(fmt.Sprintf("https://api.tenderly.co/api/v1/account/%s/project/%s", accountID, project)).
		SetHeader("X-Access-Key", token).
		SetHeader("Content-Type", "application/json").
		SetTimeout(30 * time.Second)

	return &TenderlyClient{
		client:    client,
		accountID: accountID,
		project:   project,
	}
}

// RequireTenderly skips the test if Tenderly env vars are not set.
func RequireTenderly(t testing.TB) *TenderlyClient {
	t.Helper()
	tc := NewTenderlyClient()
	if tc == nil {
		t.Skip("Tenderly env vars not set (TENDERLY_ACCESS_KEY, TENDERLY_ACCOUNT_ID, TENDERLY_PROJECT)")
	}
	return tc
}

// Simulate sends a transaction simulation to Tenderly and returns the result.
func (tc *TenderlyClient) Simulate(req SimulateRequest) (*SimulationResult, error) {
	var result SimulationResult

	resp, err := tc.client.R().
		SetBody(req).
		SetResult(&result).
		Post("/simulate")
	if err != nil {
		return nil, fmt.Errorf("tenderly simulate request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("tenderly simulate returned %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// bundleRequest wraps multiple simulations in a single API call.
type bundleRequest struct {
	Simulations []SimulateRequest `json:"simulations"`
}

// bundleResponse is the response from /simulate-bundle.
type bundleResponse struct {
	SimulationResults []SimulationResult `json:"simulation_results"`
}

// SimulateBundle sends multiple simulations in a single API call.
// Each simulation runs sequentially within the same block.
func (tc *TenderlyClient) SimulateBundle(reqs []SimulateRequest) ([]SimulationResult, error) {
	var result bundleResponse

	resp, err := tc.client.R().
		SetBody(bundleRequest{Simulations: reqs}).
		SetResult(&result).
		Post("/simulate-bundle")
	if err != nil {
		return nil, fmt.Errorf("tenderly simulate-bundle request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("tenderly simulate-bundle returned %d: %s", resp.StatusCode(), resp.String())
	}

	return result.SimulationResults, nil
}

// EstimateGas simulates a swap transaction and compares against the estimated gas.
func (tc *TenderlyClient) EstimateGas(
	chainID int,
	poolAddress string,
	calldata []byte,
	estimatedGas int64,
	opts ...SimulateOption,
) (*GasEstimateResult, error) {
	cfg := defaultSimulateConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	req := SimulateRequest{
		NetworkID:    fmt.Sprintf("%d", chainID),
		From:         cfg.sender.Hex(),
		To:           poolAddress,
		Input:        "0x" + hex.EncodeToString(calldata),
		Gas:          cfg.gas,
		GasPrice:     cfg.gasPrice.String(),
		Save:         true,
		SaveIfFails:  true,
		StateObjects: cfg.stateOverrides,
		BlockNumber:  cfg.blockNumber,
	}
	if cfg.value != nil {
		req.Value = cfg.value.String()
	}

	result, err := tc.Simulate(req)
	if err != nil {
		return nil, err
	}

	simGas := result.Simulation.GasUsed
	diff := simGas - estimatedGas
	var diffPct float64
	if simGas > 0 {
		diffPct = float64(diff) / float64(simGas) * 100
	}

	return &GasEstimateResult{
		SimulatedGas: simGas,
		EstimatedGas: estimatedGas,
		Difference:   diff,
		DiffPercent:  diffPct,
		SimulationID: result.Simulation.ID,
		TxStatus:     result.Transaction.Status,
	}, nil
}

// SimulationURL returns the Tenderly dashboard URL for a simulation.
func (tc *TenderlyClient) SimulationURL(simulationID string) string {
	return fmt.Sprintf(
		"https://dashboard.tenderly.co/%s/%s/simulator/%s",
		tc.accountID, tc.project, simulationID,
	)
}

// --- Simulate options ---

type simulateConfig struct {
	sender         common.Address
	value          *big.Int
	gas            int64
	gasPrice       *big.Int
	stateOverrides StateOverride
	blockNumber    uint64
}

// SimulateOption configures a simulation request.
type SimulateOption func(*simulateConfig)

func defaultSimulateConfig() *simulateConfig {
	return &simulateConfig{
		sender:         DefaultSender,
		gas:            1_000_000,
		gasPrice:       new(big.Int).Set(defaultGasPrice),
		stateOverrides: make(StateOverride),
	}
}

// ensureStorage returns (or creates) the storage map for a contract address.
func ensureStorage(c *simulateConfig, addr string) map[string]string {
	obj := c.stateOverrides[addr]
	if obj == nil {
		obj = &StateObject{Storage: make(map[string]string)}
		c.stateOverrides[addr] = obj
	} else if obj.Storage == nil {
		obj.Storage = make(map[string]string)
	}
	return obj.Storage
}

// WithSender sets the transaction sender address.
func WithSender(addr common.Address) SimulateOption {
	return func(c *simulateConfig) { c.sender = addr }
}

// WithValue sets the ETH value sent with the transaction.
func WithValue(v *big.Int) SimulateOption {
	return func(c *simulateConfig) { c.value = new(big.Int).Set(v) }
}

// WithGas sets the transaction gas limit.
func WithGas(v int64) SimulateOption {
	return func(c *simulateConfig) { c.gas = v }
}

// WithGasPrice sets the transaction gas price.
func WithGasPrice(v *big.Int) SimulateOption {
	return func(c *simulateConfig) { c.gasPrice = new(big.Int).Set(v) }
}

// WithBlockNumber pins the simulation to a specific block.
func WithBlockNumber(n uint64) SimulateOption {
	return func(c *simulateConfig) { c.blockNumber = n }
}

// WithStateOverrides merges additional state overrides into the request.
func WithStateOverrides(overrides StateOverride) SimulateOption {
	return func(c *simulateConfig) {
		for addr, obj := range overrides {
			c.stateOverrides[addr] = obj
		}
	}
}

// WithTokenBalance overrides a token balance for an account.
// Computes the storage slot for Solidity's mapping(address => uint256) at the given slot index.
func WithTokenBalance(token, account common.Address, balanceSlot int, amount *big.Int) SimulateOption {
	return func(c *simulateConfig) {
		slot := calcMappingSlot(big.NewInt(int64(balanceSlot)), account, false)
		storage := ensureStorage(c, strings.ToLower(token.Hex()))
		storage[slot] = common.BigToHash(amount).Hex()
	}
}

// WithTokenAllowance overrides a token allowance for owner→spender.
// Computes the storage slot for Solidity's mapping(address => mapping(address => uint256)).
func WithTokenAllowance(token, owner, spender common.Address, allowanceSlot int, amount *big.Int) SimulateOption {
	return func(c *simulateConfig) {
		slot := calcNestedMappingSlot(big.NewInt(int64(allowanceSlot)), owner, spender, false)
		storage := ensureStorage(c, strings.ToLower(token.Hex()))
		storage[slot] = common.BigToHash(amount).Hex()
	}
}

// WithRawStorageOverride sets an arbitrary storage slot on a contract.
func WithRawStorageOverride(contract common.Address, slot string, value string) SimulateOption {
	return func(c *simulateConfig) {
		storage := ensureStorage(c, strings.ToLower(contract.Hex()))
		storage[slot] = value
	}
}

// WithNativeBalance overrides the native token (ETH) balance for an account.
func WithNativeBalance(account common.Address, amount *big.Int) SimulateOption {
	return func(c *simulateConfig) {
		addr := strings.ToLower(account.Hex())
		if c.stateOverrides[addr] == nil {
			c.stateOverrides[addr] = &StateObject{}
		}
		c.stateOverrides[addr].Balance = amount.String()
	}
}

// --- Slot calculation ---
//
// Solidity: keccak256(abi.encode(key, slot))         — key first, slot second
// Vyper:    keccak256(abi.encode(slot, key))         — slot first, key second

// calcMappingSlot computes the storage key for mapping(address => T).
func calcMappingSlot(slot *big.Int, key common.Address, vyper bool) string {
	var data [64]byte
	if vyper {
		slot.FillBytes(data[0:32])
		copy(data[44:64], key.Bytes())
	} else {
		copy(data[12:32], key.Bytes())
		slot.FillBytes(data[32:64])
	}
	return crypto.Keccak256Hash(data[:]).Hex()
}

// calcNestedMappingSlot computes the storage key for mapping(address => mapping(address => T)).
func calcNestedMappingSlot(slot *big.Int, outerKey, innerKey common.Address, vyper bool) string {
	// Outer: hash(outerKey, slot) or hash(slot, outerKey) for Vyper
	var outerData [64]byte
	if vyper {
		slot.FillBytes(outerData[0:32])
		copy(outerData[44:64], outerKey.Bytes())
	} else {
		copy(outerData[12:32], outerKey.Bytes())
		slot.FillBytes(outerData[32:64])
	}
	outerHash := crypto.Keccak256(outerData[:])

	// Inner: hash(innerKey, outerHash) or hash(outerHash, innerKey) for Vyper
	var innerData [64]byte
	if vyper {
		copy(innerData[0:32], outerHash)
		copy(innerData[44:64], innerKey.Bytes())
	} else {
		copy(innerData[12:32], innerKey.Bytes())
		copy(innerData[32:64], outerHash)
	}
	return crypto.Keccak256Hash(innerData[:]).Hex()
}

// --- Batch simulation ---

// SwapCase describes a single swap to simulate for gas comparison.
type SwapCase struct {
	Name         string
	PoolAddress  string
	Calldata     []byte
	EstimatedGas int64
	Options      []SimulateOption
}

// BatchEstimateGas runs multiple swap simulations and logs results.
func (tc *TenderlyClient) BatchEstimateGas(
	t *testing.T,
	chainID int,
	cases []SwapCase,
) {
	t.Helper()
	for _, sc := range cases {
		t.Run(sc.Name, func(t *testing.T) {
			t.Helper()
			result, err := tc.EstimateGas(chainID, sc.PoolAddress, sc.Calldata, sc.EstimatedGas, sc.Options...)
			if err != nil {
				t.Fatalf("simulation failed: %v", err)
			}

			url := tc.SimulationURL(result.SimulationID)
			t.Logf("Simulation: %s", url)
			t.Logf("  Status:     %v", result.TxStatus)
			t.Logf("  Simulated:  %d gas", result.SimulatedGas)
			t.Logf("  Estimated:  %d gas", result.EstimatedGas)
			t.Logf("  Difference: %d (%.1f%%)", result.Difference, result.DiffPercent)

			if !result.TxStatus {
				t.Errorf("transaction reverted — check %s", url)
			}
		})
	}
}

// --- Calldata encoding ---

// sigPattern extracts function name and param types from a Solidity signature.
var sigPattern = regexp.MustCompile(`^(\w+)\(([^)]*)\)$`)

// EncodeSwapCalldata ABI-encodes a function call from its Solidity signature and arguments.
// Only flat parameter lists are supported — tuple types like (address,uint256) are not.
//
// Example:
//
//	calldata, _ := EncodeSwapCalldata("swap(uint256,bool)", big.NewInt(1e18), true)
//	calldata, _ := EncodeSwapCalldata("swap(uint256,uint256,address,bytes)", amountIn, amountOutMin, to, []byte{})
func EncodeSwapCalldata(signature string, args ...any) ([]byte, error) {
	matches := sigPattern.FindStringSubmatch(signature)
	if matches == nil {
		return nil, fmt.Errorf("invalid function signature: %s", signature)
	}

	funcName := matches[1]
	paramStr := strings.TrimSpace(matches[2])

	if strings.Contains(paramStr, "(") {
		return nil, fmt.Errorf("tuple parameter types are not supported: %s", signature)
	}

	var argTypes abi.Arguments
	if paramStr != "" {
		for _, typeStr := range strings.Split(paramStr, ",") {
			typeStr = strings.TrimSpace(typeStr)
			abiType, err := abi.NewType(typeStr, "", nil)
			if err != nil {
				return nil, fmt.Errorf("invalid ABI type %q: %w", typeStr, err)
			}
			argTypes = append(argTypes, abi.Argument{Type: abiType})
		}
	}

	method := abi.NewMethod(funcName, funcName, abi.Function, "", false, false, argTypes, nil)
	packed, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, fmt.Errorf("packing args for %s: %w", signature, err)
	}

	return append(method.ID, packed...), nil
}

// --- Debug helpers ---

// DumpSimulateRequest returns a pretty-printed JSON of the simulation request.
func DumpSimulateRequest(req SimulateRequest) string {
	b, err := gojson.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

// MaxUint256 returns the maximum uint256 value, useful for unlimited approvals.
func MaxUint256() *big.Int {
	return new(big.Int).Set(maxUint256)
}

// --- ERC20 token storage slot discovery ---

// ERC7201 base slot for OpenZeppelin ERC20Upgradeable:
// keccak256(abi.encode(uint256(keccak256("openzeppelin.storage.ERC20")) - 1)) & ~0xff
var erc7201BaseSlot = func() *big.Int {
	v, ok := new(big.Int).SetString("52c63247e1f47db19d5ce0460030c497f067ca4cebf71ba98eeadabe20bace00", 16)
	if !ok {
		panic("tenderly: invalid erc7201BaseSlot hex constant")
	}
	return v
}()

// slotSearchRanges are the starting points and range to brute-force for mapping slots.
// Covers standard contracts (0..999) and ERC7201 upgradeable (base..base+999).
var slotSearchRanges = [][2]*big.Int{
	{big.NewInt(0), big.NewInt(1000)},
	{new(big.Int).Set(erc7201BaseSlot), new(big.Int).Add(new(big.Int).Set(erc7201BaseSlot), big.NewInt(1000))},
}

var knownTokenSlots = map[int]map[string]TokenStorageSlots{
	4326: {
		strings.ToLower("0x4200000000000000000000000000000000000006"): {
			BalanceSlot: big.NewInt(3),
			AllowSlot:   big.NewInt(4),
		},
		strings.ToLower("0xB0F70C0bD6FD87dbEb7C10dC692a2a6106817072"): {
			BalanceSlot: new(big.Int).Set(erc7201BaseSlot),
			AllowSlot:   new(big.Int).Add(new(big.Int).Set(erc7201BaseSlot), big.NewInt(1)),
		},
		strings.ToLower("0xFAfDdbb3FC7688494971a79cc65DCa3EF82079E7"): {
			BalanceSlot: new(big.Int).Set(erc7201BaseSlot),
			AllowSlot:   new(big.Int).Add(new(big.Int).Set(erc7201BaseSlot), big.NewInt(1)),
		},
		strings.ToLower("0xB8CE59FC3717ada4C02eaDF9682A9e934F625ebb"): {
			BalanceSlot: new(big.Int).Set(erc7201BaseSlot),
			AllowSlot:   new(big.Int).Add(new(big.Int).Set(erc7201BaseSlot), big.NewInt(1)),
		},
	},
}

// TokenStorageSlots holds the discovered balance and allowance mapping slots for a token.
type TokenStorageSlots struct {
	BalanceSlot *big.Int
	AllowSlot   *big.Int
	IsVyper     bool
	StateProxy  string // non-empty if storage lives on a proxy contract
}

func lookupKnownTokenSlots(chainID int, token common.Address) (*TokenStorageSlots, bool) {
	chainSlots, ok := knownTokenSlots[chainID]
	if !ok {
		return nil, false
	}

	slots, ok := chainSlots[strings.ToLower(token.Hex())]
	if !ok {
		return nil, false
	}

	return &TokenStorageSlots{
		BalanceSlot: cloneBigInt(slots.BalanceSlot),
		AllowSlot:   cloneBigInt(slots.AllowSlot),
		IsVyper:     slots.IsVyper,
		StateProxy:  slots.StateProxy,
	}, true
}

func cloneBigInt(v *big.Int) *big.Int {
	if v == nil {
		return nil
	}
	return new(big.Int).Set(v)
}

// searchSlotRanges iterates over all candidate slots and calls match for each (candidate, isVyper).
// Returns the first match. Reuses big.Int allocations to reduce GC pressure.
func searchSlotRanges(match func(candidate *big.Int, vyper bool) bool) (*big.Int, bool, bool) {
	var candidate big.Int
	for _, r := range slotSearchRanges {
		candidate.Set(r[0])
		for candidate.Cmp(r[1]) < 0 {
			if match(&candidate, false) {
				return new(big.Int).Set(&candidate), false, true
			}
			if match(&candidate, true) {
				return new(big.Int).Set(&candidate), true, true
			}
			candidate.Add(&candidate, big.NewInt(1))
		}
	}
	return nil, false, false
}

// --- Tenderly trace-based slot discovery ---

type simulationDetails struct {
	Transaction struct {
		TransactionInfo struct {
			CallTrace callTrace `json:"call_trace"`
		} `json:"transaction_info"`
	} `json:"transaction"`
}

type callTrace struct {
	CallType       string      `json:"call_type"`
	Output         string      `json:"output"`
	StorageAddress string      `json:"storage_address"`
	StorageSlot    []string    `json:"storage_slot"`
	Calls          []callTrace `json:"calls"`
}

type readSlot struct {
	Slot           string
	Address        string
	ParentCallType string
}

// getSLOADSlots recursively extracts all SLOAD operations from a call trace.
func getSLOADSlots(ct callTrace, parentCallType string) []readSlot {
	var results []readSlot
	if ct.CallType == "SLOAD" && len(ct.StorageSlot) > 0 {
		results = append(results, readSlot{
			Slot:           ct.StorageSlot[0],
			Address:        ct.StorageAddress,
			ParentCallType: parentCallType,
		})
	}
	for _, child := range ct.Calls {
		results = append(results, getSLOADSlots(child, ct.CallType)...)
	}
	return results
}

// indexReadSlots builds a map[slotHex]readSlot for O(1) lookup.
func indexReadSlots(slots []readSlot) map[string]readSlot {
	m := make(map[string]readSlot, len(slots))
	for _, rs := range slots {
		m[rs.Slot] = rs
	}
	return m
}

// getSimulationDetails fetches the full trace for a simulation ID.
func (tc *TenderlyClient) getSimulationDetails(simID string) (*simulationDetails, error) {
	var result simulationDetails
	resp, err := tc.client.R().
		SetResult(&result).
		Get("/simulations/" + simID)
	if err != nil {
		return nil, fmt.Errorf("get simulation details: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("get simulation details returned %d: %s", resp.StatusCode(), resp.String())
	}
	return &result, nil
}

// simulateViewCall simulates a read-only call and returns the SLOAD trace.
// Uses simulation_type="full" to get the call trace in a single API call.
// Falls back to a second API call if the trace is empty (older API versions).
func (tc *TenderlyClient) simulateViewCall(chainID int, token string, calldata []byte) ([]readSlot, error) {
	req := SimulateRequest{
		NetworkID:      fmt.Sprintf("%d", chainID),
		From:           DefaultSender.Hex(),
		To:             token,
		Input:          "0x" + hex.EncodeToString(calldata),
		Gas:            100000000,
		GasPrice:       defaultGasPrice.String(),
		Save:           true,
		SaveIfFails:    true,
		SimulationType: "full",
		StateObjects: StateOverride{
			defaultSenderLower: {Balance: "1000000000000000000"},
		},
	}

	result, err := tc.Simulate(req)
	if err != nil {
		return nil, err
	}

	// Try trace from simulate response first (1 call)
	slots := getSLOADSlots(result.Transaction.TransactionInfo.CallTrace, "")
	if len(slots) > 0 {
		return slots, nil
	}

	// Fallback: fetch trace separately (2nd call)
	details, err := tc.getSimulationDetails(result.Simulation.ID)
	if err != nil {
		return nil, err
	}

	return getSLOADSlots(details.Transaction.TransactionInfo.CallTrace, ""), nil
}

func decodeUint256Hex(output string) (*big.Int, error) {
	output = strings.TrimPrefix(output, "0x")
	if output == "" {
		return big.NewInt(0), nil
	}

	v, ok := new(big.Int).SetString(output, 16)
	if !ok {
		return nil, fmt.Errorf("invalid uint256 hex output %q", output)
	}

	return v, nil
}

func (tc *TenderlyClient) simulateViewUint256(
	chainID int,
	token common.Address,
	calldata []byte,
	overrides StateOverride,
) (*big.Int, error) {
	req := SimulateRequest{
		NetworkID:    fmt.Sprintf("%d", chainID),
		From:         DefaultSender.Hex(),
		To:           token.Hex(),
		Input:        "0x" + hex.EncodeToString(calldata),
		Gas:          100_000,
		GasPrice:     defaultGasPrice.String(),
		Save:         true,
		SaveIfFails:  true,
		StateObjects: overrides,
	}

	result, err := tc.Simulate(req)
	if err != nil {
		return nil, err
	}
	if !result.Transaction.Status {
		return nil, fmt.Errorf("simulation reverted")
	}

	return decodeUint256Hex(result.Transaction.TransactionInfo.CallTrace.Output)
}

// traceGuidedSlotSearch tests SLOAD slots from the trace directly by overriding each with a
// non-zero value and checking if the function output changes. This handles proxy tokens with
// custom ERC7201 namespaces where the base slot is not in our predefined search ranges.
// For each SLOAD that changes the output, it reverse-computes the base mapping slot.
func (tc *TenderlyClient) traceGuidedSlotSearch(
	chainID int,
	token common.Address,
	calldata []byte,
	readSlots []readSlot,
	computeSlot func(candidate *big.Int, vyper bool) string,
	setResult func(*TokenStorageSlots, *big.Int, bool),
) *TokenStorageSlots {
	if len(readSlots) == 0 {
		return nil
	}

	networkID := fmt.Sprintf("%d", chainID)
	inputHex := "0x" + hex.EncodeToString(calldata)
	tokenLower := strings.ToLower(token.Hex())
	// Use a large sentinel that survives struct packing (upper bits non-zero)
	sentinel := new(big.Int).Lsh(big.NewInt(999999999), 128)

	// Collect unique SLOAD hashes (excluding well-known non-mapping slots like ERC1967 impl slot)
	erc1967Impl := "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
	var traceSlots []string
	seen := make(map[string]bool)
	for _, rs := range readSlots {
		slot := rs.Slot
		if seen[slot] || slot == erc1967Impl {
			continue
		}
		seen[slot] = true
		traceSlots = append(traceSlots, slot)
	}

	// Build a bundle: one simulation per trace SLOAD, overriding that slot with sentinel
	reqs := make([]SimulateRequest, len(traceSlots))
	for i, slot := range traceSlots {
		// Override on the token address (proxy storage lives at proxy address)
		reqs[i] = SimulateRequest{
			NetworkID:      networkID,
			From:           DefaultSender.Hex(),
			To:             token.Hex(),
			Input:          inputHex,
			Gas:            100_000,
			GasPrice:       defaultGasPrice.String(),
			Save:           false,
			SaveIfFails:    false,
			SimulationType: "quick",
			StateObjects: StateOverride{
				defaultSenderLower: {Balance: "1000000000000000000"},
				tokenLower: {
					Storage: map[string]string{
						slot: common.BigToHash(sentinel).Hex(),
					},
				},
			},
		}
	}

	results, err := tc.SimulateBundle(reqs)
	if err != nil {
		return nil
	}

	for i, res := range results {
		if !res.Transaction.Status {
			continue
		}
		out, err := decodeUint256Hex(res.Transaction.TransactionInfo.CallTrace.Output)
		if err != nil || out.Sign() == 0 {
			continue
		}

		// This trace SLOAD is the balance slot. Now reverse-compute the base mapping slot.
		// traceSlots[i] = keccak256(abi.encode(sender, baseSlot)) for Solidity
		// We can't reverse keccak256, but we can search our ranges for a base that produces this hash.
		matchedSlot := traceSlots[i]

		// Search standard ranges first
		base, vyper, found := searchSlotRanges(func(candidate *big.Int, v bool) bool {
			return computeSlot(candidate, v) == matchedSlot
		})
		if found {
			result := &TokenStorageSlots{IsVyper: vyper}
			setResult(result, base, vyper)
			return result
		}

		// Search common ERC7201 namespaces
		namespaces := []string{
			"openzeppelin.storage.ERC20",
			"AgoraDollarErc1967Proxy.Erc20CoreStorage",
		}
		for _, ns := range namespaces {
			nsHash := crypto.Keccak256([]byte(ns))
			nsVal := new(big.Int).SetBytes(nsHash)
			nsVal.Sub(nsVal, big.NewInt(1))
			encoded := common.LeftPadBytes(nsVal.Bytes(), 32)
			bh := crypto.Keccak256(encoded)
			bh[31] = 0
			baseSlot := new(big.Int).SetBytes(bh)

			for offset := int64(0); offset < 10; offset++ {
				candidate := new(big.Int).Add(baseSlot, big.NewInt(offset))
				for _, v := range []bool{false, true} {
					if computeSlot(candidate, v) == matchedSlot {
						result := &TokenStorageSlots{IsVyper: v}
						setResult(result, candidate, v)
						return result
					}
				}
			}
		}

	}

	return nil
}

// slotCandidate pairs a slot index with a vyper flag for batch testing.
type slotCandidate struct {
	slot  *big.Int
	vyper bool
}

func (tc *TenderlyClient) bruteForceTokenSlot(
	chainID int,
	token common.Address,
	calldata []byte,
	computeSlot func(candidate *big.Int, vyper bool) string,
	setResult func(*TokenStorageSlots, *big.Int, bool),
) (*TokenStorageSlots, error) {
	sentinel := big.NewInt(123456789)
	tokenLower := strings.ToLower(token.Hex())
	networkID := fmt.Sprintf("%d", chainID)
	inputHex := "0x" + hex.EncodeToString(calldata)

	// testGroup runs a single simulation with ALL candidates' storage keys set to sentinel.
	// If balanceOf returns sentinel, one of the candidates is correct.
	testGroup := func(candidates []slotCandidate) (bool, error) {
		if len(candidates) == 0 {
			return false, nil
		}
		storage := make(map[string]string, len(candidates))
		for _, c := range candidates {
			storage[computeSlot(c.slot, c.vyper)] = common.BigToHash(sentinel).Hex()
		}
		req := SimulateRequest{
			NetworkID:      networkID,
			From:           DefaultSender.Hex(),
			To:             token.Hex(),
			Input:          inputHex,
			Gas:            100_000,
			GasPrice:       defaultGasPrice.String(),
			Save:           false,
			SaveIfFails:    false,
			SimulationType: "quick",
			StateObjects: StateOverride{
				defaultSenderLower: {Balance: "1000000000000000000"},
				tokenLower:         {Storage: storage},
			},
		}
		result, err := tc.Simulate(req)
		if err != nil {
			return false, err
		}
		if !result.Transaction.Status {
			return false, nil
		}
		out, err := decodeUint256Hex(result.Transaction.TransactionInfo.CallTrace.Output)
		if err != nil {
			return false, nil
		}
		return out.Cmp(sentinel) == 0, nil
	}

	// binarySearch finds the exact matching candidate in O(log n) API calls.
	var binarySearch func(candidates []slotCandidate) (*slotCandidate, error)
	binarySearch = func(candidates []slotCandidate) (*slotCandidate, error) {
		if len(candidates) == 0 {
			return nil, nil
		}
		if len(candidates) == 1 {
			hit, err := testGroup(candidates)
			if err != nil {
				return nil, err
			}
			if hit {
				return &candidates[0], nil
			}
			return nil, nil
		}
		mid := len(candidates) / 2
		// Test left half first
		hit, err := testGroup(candidates[:mid])
		if err != nil {
			return nil, err
		}
		if hit {
			return binarySearch(candidates[:mid])
		}
		// Try right half
		hit, err = testGroup(candidates[mid:])
		if err != nil {
			return nil, err
		}
		if hit {
			return binarySearch(candidates[mid:])
		}
		return nil, nil
	}

	// Collect all candidates: prioritized first, then full ranges.
	// Pre-allocate: (21+11)*2 prioritized + (979+989)*2 full ranges ≈ 4000 entries
	allCandidates := make([]slotCandidate, 0, 4000)

	// Prioritized: slots 0-20 + ERC7201 base+0..+10, both Solidity and Vyper.
	for i := int64(0); i <= 20; i++ {
		allCandidates = append(allCandidates,
			slotCandidate{big.NewInt(i), false},
			slotCandidate{big.NewInt(i), true},
		)
	}
	for i := int64(0); i <= 10; i++ {
		s := new(big.Int).Add(new(big.Int).Set(erc7201BaseSlot), big.NewInt(i))
		allCandidates = append(allCandidates,
			slotCandidate{s, false},
			slotCandidate{s, true},
		)
	}

	// Full ranges: 21-999 + ERC7201+11..+999, both layouts.
	for _, r := range slotSearchRanges {
		var candidate big.Int
		candidate.Set(r[0])
		for candidate.Cmp(r[1]) < 0 {
			// Skip already-covered prioritized slots.
			if candidate.Cmp(big.NewInt(20)) <= 0 && r[0].Sign() == 0 {
				candidate.Add(&candidate, big.NewInt(1))
				continue
			}
			if r[0].Cmp(erc7201BaseSlot) == 0 {
				offset := new(big.Int).Sub(&candidate, erc7201BaseSlot)
				if offset.Cmp(big.NewInt(10)) <= 0 {
					candidate.Add(&candidate, big.NewInt(1))
					continue
				}
			}
			allCandidates = append(allCandidates,
				slotCandidate{new(big.Int).Set(&candidate), false},
				slotCandidate{new(big.Int).Set(&candidate), true},
			)
			candidate.Add(&candidate, big.NewInt(1))
		}
	}

	// First: test ALL candidates at once to confirm the slot exists in our search space.
	hit, err := testGroup(allCandidates)
	if err != nil {
		return nil, fmt.Errorf("brute-force initial test failed: %w", err)
	}
	if !hit {
		return nil, fmt.Errorf("could not find matching storage slot for token %s on chain %d", token.Hex(), chainID)
	}

	// Binary search to find the exact candidate: O(log n) API calls.
	match, err := binarySearch(allCandidates)
	if err != nil {
		return nil, fmt.Errorf("brute-force binary search failed: %w", err)
	}
	if match != nil {
		result := &TokenStorageSlots{IsVyper: match.vyper}
		setResult(result, match.slot, match.vyper)
		return result, nil
	}

	return nil, fmt.Errorf("could not find matching storage slot for token %s on chain %d", token.Hex(), chainID)
}

// findSlotInTrace searches trace readSlots for a matching candidate using the given slot calculator.
// Returns the matching TokenStorageSlots or nil if not found.
func findSlotInTrace(
	slotIndex map[string]readSlot,
	tokenLower string,
	computeSlot func(candidate *big.Int, vyper bool) string,
	setResult func(slots *TokenStorageSlots, candidate *big.Int, vyper bool),
) *TokenStorageSlots {
	slot, vyper, found := searchSlotRanges(func(candidate *big.Int, vyper bool) bool {
		key := computeSlot(candidate, vyper)
		_, ok := slotIndex[key]
		return ok
	})
	if !found {
		return nil
	}

	result := &TokenStorageSlots{IsVyper: vyper}
	setResult(result, slot, vyper)

	// Check proxy
	key := computeSlot(slot, vyper)
	if rs, ok := slotIndex[key]; ok {
		if rs.Address != tokenLower && rs.ParentCallType != "DELEGATECALL" {
			result.StateProxy = rs.Address
		}
	}

	return result
}

// FindBalanceOfSlot discovers the storage slot index of a token's balanceOf mapping
// by simulating a balanceOf() call and analyzing SLOAD traces.
func (tc *TenderlyClient) FindBalanceOfSlot(chainID int, token common.Address) (*TokenStorageSlots, error) {
	if slots, ok := lookupKnownTokenSlots(chainID, token); ok && slots.BalanceSlot != nil {
		return &TokenStorageSlots{
			BalanceSlot: cloneBigInt(slots.BalanceSlot),
			IsVyper:     slots.IsVyper,
			StateProxy:  slots.StateProxy,
		}, nil
	}

	calldata, err := EncodeSwapCalldata("balanceOf(address)", DefaultSender)
	if err != nil {
		return nil, fmt.Errorf("encode balanceOf: %w", err)
	}

	readSlots, err := tc.simulateViewCall(chainID, token.Hex(), calldata)
	if err != nil {
		return nil, fmt.Errorf("simulate balanceOf: %w", err)
	}

	slotIndex := indexReadSlots(readSlots)
	tokenLower := strings.ToLower(token.Hex())

	computeBalSlot := func(candidate *big.Int, vyper bool) string {
		return calcMappingSlot(candidate, DefaultSender, vyper)
	}
	setBalSlot := func(s *TokenStorageSlots, candidate *big.Int, _ bool) {
		s.BalanceSlot = candidate
	}

	result := findSlotInTrace(slotIndex, tokenLower, computeBalSlot, setBalSlot)
	if result != nil {
		return result, nil
	}

	// Trace-guided: test each SLOAD from the trace with a sentinel override
	result = tc.traceGuidedSlotSearch(chainID, token, calldata, readSlots, computeBalSlot, setBalSlot)
	if result != nil {
		return result, nil
	}

	result, err = tc.bruteForceTokenSlot(chainID, token, calldata, computeBalSlot, setBalSlot)
	if err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("could not find balanceOf mapping slot for token %s on chain %d", token.Hex(), chainID)
}

// FindAllowanceSlot discovers the storage slot index of a token's allowance mapping
// by simulating an allowance() call and analyzing SLOAD traces.
func (tc *TenderlyClient) FindAllowanceSlot(chainID int, token common.Address) (*TokenStorageSlots, error) {
	if slots, ok := lookupKnownTokenSlots(chainID, token); ok && slots.AllowSlot != nil {
		return &TokenStorageSlots{
			AllowSlot:  cloneBigInt(slots.AllowSlot),
			IsVyper:    slots.IsVyper,
			StateProxy: slots.StateProxy,
		}, nil
	}

	calldata, err := EncodeSwapCalldata("allowance(address,address)", DefaultSender, DefaultSpender)
	if err != nil {
		return nil, fmt.Errorf("encode allowance: %w", err)
	}

	readSlots, err := tc.simulateViewCall(chainID, token.Hex(), calldata)
	if err != nil {
		return nil, fmt.Errorf("simulate allowance: %w", err)
	}

	slotIndex := indexReadSlots(readSlots)
	tokenLower := strings.ToLower(token.Hex())

	computeAllowSlot := func(candidate *big.Int, vyper bool) string {
		return calcNestedMappingSlot(candidate, DefaultSender, DefaultSpender, vyper)
	}
	setAllowSlot := func(s *TokenStorageSlots, candidate *big.Int, _ bool) {
		s.AllowSlot = candidate
	}

	result := findSlotInTrace(slotIndex, tokenLower, computeAllowSlot, setAllowSlot)
	if result != nil {
		return result, nil
	}

	// Trace-guided: test each SLOAD from the trace with a sentinel override
	result = tc.traceGuidedSlotSearch(chainID, token, calldata, readSlots, computeAllowSlot, setAllowSlot)
	if result != nil {
		return result, nil
	}

	result, err = tc.bruteForceTokenSlot(chainID, token, calldata, computeAllowSlot, setAllowSlot)
	if err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("could not find allowance mapping slot for token %s on chain %d", token.Hex(), chainID)
}

// FindTokenSlots discovers both balanceOf and allowance mapping slots in a single bundled API call.
func (tc *TenderlyClient) FindTokenSlots(chainID int, token common.Address) (*TokenStorageSlots, error) {
	balCalldata, err := EncodeSwapCalldata("balanceOf(address)", DefaultSender)
	if err != nil {
		return nil, fmt.Errorf("encode balanceOf: %w", err)
	}

	allowCalldata, err := EncodeSwapCalldata("allowance(address,address)", DefaultSender, DefaultSpender)
	if err != nil {
		return nil, fmt.Errorf("encode allowance: %w", err)
	}

	tokenHex := token.Hex()
	baseSim := SimulateRequest{
		NetworkID:      fmt.Sprintf("%d", chainID),
		From:           DefaultSender.Hex(),
		To:             tokenHex,
		Gas:            100_000,
		GasPrice:       defaultGasPrice.String(),
		Save:           true,
		SaveIfFails:    true,
		SimulationType: "full",
		StateObjects: StateOverride{
			defaultSenderLower: {Balance: "1000000000000000000"},
		},
	}

	balSim := baseSim
	balSim.Input = "0x" + hex.EncodeToString(balCalldata)
	allowSim := baseSim
	allowSim.Input = "0x" + hex.EncodeToString(allowCalldata)

	// 1 API call for both
	results, err := tc.SimulateBundle([]SimulateRequest{balSim, allowSim})
	if err != nil {
		return nil, fmt.Errorf("simulate bundle: %w", err)
	}
	if len(results) < 2 {
		return nil, fmt.Errorf("bundle returned %d results, expected 2", len(results))
	}

	tokenLower := strings.ToLower(tokenHex)

	// Parse balance slot from first simulation trace
	balSlots := getSLOADSlots(results[0].Transaction.TransactionInfo.CallTrace, "")
	balIndex := indexReadSlots(balSlots)
	balResult := findSlotInTrace(balIndex, tokenLower,
		func(candidate *big.Int, vyper bool) string {
			return calcMappingSlot(candidate, DefaultSender, vyper)
		},
		func(s *TokenStorageSlots, candidate *big.Int, _ bool) {
			s.BalanceSlot = candidate
		},
	)
	if balResult == nil {
		return nil, fmt.Errorf("could not find balanceOf mapping slot for token %s on chain %d", tokenHex, chainID)
	}

	// Parse allowance slot from second simulation trace
	allowSlots := getSLOADSlots(results[1].Transaction.TransactionInfo.CallTrace, "")
	allowIndex := indexReadSlots(allowSlots)
	allowResult := findSlotInTrace(allowIndex, tokenLower,
		func(candidate *big.Int, vyper bool) string {
			return calcNestedMappingSlot(candidate, DefaultSender, DefaultSpender, vyper)
		},
		func(s *TokenStorageSlots, candidate *big.Int, _ bool) {
			s.AllowSlot = candidate
		},
	)
	if allowResult == nil {
		return nil, fmt.Errorf("could not find allowance mapping slot for token %s on chain %d", tokenHex, chainID)
	}

	return &TokenStorageSlots{
		BalanceSlot: balResult.BalanceSlot,
		AllowSlot:   allowResult.AllowSlot,
		IsVyper:     balResult.IsVyper,
		StateProxy:  balResult.StateProxy,
	}, nil
}

// --- Convenience: auto-discovered overrides ---

// WithDiscoveredTokenBalance overrides a token balance using auto-discovered storage slots.
// Slower than WithTokenBalance (requires 1 Tenderly simulation) but works for any standard token.
func (tc *TenderlyClient) WithDiscoveredTokenBalance(chainID int, token, account common.Address, amount *big.Int) (SimulateOption, error) {
	slots, err := tc.FindBalanceOfSlot(chainID, token)
	if err != nil {
		return nil, err
	}

	computedSlot := calcMappingSlot(slots.BalanceSlot, account, slots.IsVyper)
	contract := token
	if slots.StateProxy != "" {
		contract = common.HexToAddress(slots.StateProxy)
	}

	return WithRawStorageOverride(contract, computedSlot, common.BigToHash(amount).Hex()), nil
}

// WithDiscoveredTokenAllowance overrides a token allowance using auto-discovered storage slots.
// Slower than WithTokenAllowance (requires 1 Tenderly simulation) but works for any standard token.
func (tc *TenderlyClient) WithDiscoveredTokenAllowance(chainID int, token, owner, spender common.Address, amount *big.Int) (SimulateOption, error) {
	slots, err := tc.FindAllowanceSlot(chainID, token)
	if err != nil {
		return nil, err
	}

	computedSlot := calcNestedMappingSlot(slots.AllowSlot, owner, spender, slots.IsVyper)
	contract := token
	if slots.StateProxy != "" {
		contract = common.HexToAddress(slots.StateProxy)
	}

	return WithRawStorageOverride(contract, computedSlot, common.BigToHash(amount).Hex()), nil
}
