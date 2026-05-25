package lunarbase

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	stateUpdatedStaleDuration = 6 * time.Second
	reservesStaleDuration     = 10 * time.Second

	heartbeatInterval = 200 * time.Millisecond
	blockStaleThd     = 8 * time.Second
	flashStaleThd     = 800 * time.Millisecond

	reconnectBaseWait    = 500 * time.Millisecond
	reconnectMaxWait     = 30 * time.Second
	maxReconnectAttempts = 10

	dedupRingSize = 256
)

type poolState struct {
	SqrtPriceX96      *uint256.Int
	FeeAskX24         uint32
	FeeBidX24         uint32
	ReserveX          *uint256.Int
	ReserveY          *uint256.Int
	LatestUpdateBlock uint64
	BlockDelay        uint64
	ConcentrationK    uint32
	BlockNumber       uint64

	StateUpdatedAt    time.Time
	ReservesUpdatedAt time.Time
}

func (s *poolState) IsStale() bool {
	now := time.Now()
	return now.Sub(s.StateUpdatedAt) > stateUpdatedStaleDuration ||
		now.Sub(s.ReservesUpdatedAt) > reservesStaleDuration
}

type FlashBlockSubscriber struct {
	mu          sync.RWMutex
	latestState *poolState

	wsURL       string
	flashWsURL  string
	coreAddress common.Address

	lastBlockTime    atomic.Int64
	lastFlashTime    atomic.Int64
	lastEventTime    atomic.Int64
	flashFeedEnabled atomic.Bool

	dedupMu   sync.Mutex
	dedupRing [dedupRingSize]uint64
	dedupIdx  int

	forceClose chan struct{}

	cancel context.CancelFunc
}

var (
	subscriberOnce     sync.Once
	subscriberInstance *FlashBlockSubscriber
)

func InitFlashBlockSubscriber(wsURL, flashWsURL string, coreAddress common.Address) {
	subscriberOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		subscriberInstance = &FlashBlockSubscriber{
			wsURL:       wsURL,
			flashWsURL:  flashWsURL,
			coreAddress: coreAddress,
			forceClose:  make(chan struct{}, 1),
			cancel:      cancel,
		}
		go subscriberInstance.run(ctx)
	})
}

func GetFlashBlockSubscriber() *FlashBlockSubscriber {
	return subscriberInstance
}

func (s *FlashBlockSubscriber) GetLatestState() *poolState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latestState == nil {
		return nil
	}

	cp := *s.latestState
	if s.latestState.SqrtPriceX96 != nil {
		cp.SqrtPriceX96 = new(uint256.Int).Set(s.latestState.SqrtPriceX96)
	}
	cp.ReserveX = new(uint256.Int).Set(s.latestState.ReserveX)
	cp.ReserveY = new(uint256.Int).Set(s.latestState.ReserveY)

	return &cp
}

func (s *FlashBlockSubscriber) hasFlash() bool {
	return s.flashWsURL != ""
}

func (s *FlashBlockSubscriber) run(ctx context.Context) {
	attempt := 0
	for {
		if ctx.Err() != nil {
			return
		}

		err := s.connectAndListen(ctx)
		if ctx.Err() != nil {
			return
		}
		_ = err

		attempt++
		if attempt > maxReconnectAttempts {
			attempt = 1
		}

		wait := reconnectBaseWait
		for i := 1; i < attempt; i++ {
			wait *= 2
			if wait > reconnectMaxWait {
				wait = reconnectMaxWait
				break
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (s *FlashBlockSubscriber) connectAndListen(ctx context.Context) error {
	headWSURL := s.wsURL
	if headWSURL == "" {
		headWSURL = s.flashWsURL
	}
	logWSURL := s.wsURL
	if s.hasFlash() {
		logWSURL = s.flashWsURL
	}

	if headWSURL == "" || logWSURL == "" {
		return fmt.Errorf("no WebSocket URL configured")
	}

	headsClient, err := rpc.DialContext(ctx, headWSURL)
	if err != nil {
		return err
	}
	defer headsClient.Close()

	logsClient := headsClient
	if logWSURL != headWSURL {
		logsClient, err = rpc.DialContext(ctx, logWSURL)
		if err != nil {
			return err
		}
		defer logsClient.Close()
	}

	now := time.Now().UnixMilli()
	s.lastBlockTime.Store(now)
	s.lastEventTime.Store(now)
	s.flashFeedEnabled.Store(false)

	select {
	case <-s.forceClose:
	default:
	}

	headsCh := make(chan *types.Header, 16)
	headsSub, err := subscribeNewHeads(ctx, headsClient, headsCh)
	if err != nil {
		return fmt.Errorf("subscribe newHeads: %w", err)
	}
	defer headsSub.Unsubscribe()

	topicNames := []struct {
		topic common.Hash
		name  string
	}{
		{topicStateUpdated, "StateUpdated"},
		{topicSync, "Sync"},
		{topicSwapExecuted, "SwapExecuted"},
		{topicConcentrationKSet, "ConcentrationKSet"},
		{topicBlockDelaySet, "BlockDelaySet"},
	}

	subs, logsMethod, err := subscribePoolLogs(ctx, logsClient, s.coreAddress, topicNames)
	if err != nil {
		return err
	}
	defer func() {
		for _, ls := range subs {
			ls.sub.Unsubscribe()
		}
	}()

	s.flashFeedEnabled.Store(s.hasFlash() && logsMethod == "pendingLogs")
	if s.flashFeedEnabled.Load() {
		s.lastFlashTime.Store(now)
	}

	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go s.heartbeat(heartbeatCtx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-s.forceClose:
			return fmt.Errorf("heartbeat forced reconnect")

		case header := <-headsCh:
			if header != nil {
				s.lastBlockTime.Store(time.Now().UnixMilli())
				s.lastEventTime.Store(time.Now().UnixMilli())
				s.mu.Lock()
				if s.latestState != nil {
					s.latestState.BlockNumber = header.Number.Uint64()
				}
				s.mu.Unlock()
			}
		case err := <-headsSub.Err():
			return fmt.Errorf("newHeads subscription error: %w", err)

		case log := <-subs[0].ch:
			s.processLog(log)
		case err := <-subs[0].sub.Err():
			return fmt.Errorf("stateUpdated subscription error: %w", err)

		case log := <-subs[1].ch:
			s.processLog(log)
		case err := <-subs[1].sub.Err():
			return fmt.Errorf("sync subscription error: %w", err)

		case log := <-subs[2].ch:
			s.processLog(log)
		case err := <-subs[2].sub.Err():
			return fmt.Errorf("swapExecuted subscription error: %w", err)

		case log := <-subs[3].ch:
			s.processLog(log)
		case err := <-subs[3].sub.Err():
			return fmt.Errorf("concentrationKSet subscription error: %w", err)

		case log := <-subs[4].ch:
			s.processLog(log)
		case err := <-subs[4].sub.Err():
			return fmt.Errorf("blockDelaySet subscription error: %w", err)
		}
	}
}

func (s *FlashBlockSubscriber) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UnixMilli()

			if lastBlock := s.lastBlockTime.Load(); lastBlock > 0 && now-lastBlock > blockStaleThd.Milliseconds() {
				s.lastBlockTime.Store(0)
				select {
				case s.forceClose <- struct{}{}:
				default:
				}
				return
			}

			if s.flashFeedEnabled.Load() {
				if lastFlash := s.lastFlashTime.Load(); lastFlash > 0 && now-lastFlash > flashStaleThd.Milliseconds() {
					s.lastFlashTime.Store(0)
					select {
					case s.forceClose <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}
}

func (s *FlashBlockSubscriber) isDuplicate(log types.Log) bool {
	key := log.BlockNumber*1_000_000 + uint64(log.TxIndex)*1_000 + uint64(log.Index)

	s.dedupMu.Lock()
	defer s.dedupMu.Unlock()

	for _, v := range s.dedupRing {
		if v == key && key != 0 {
			return true
		}
	}

	s.dedupRing[s.dedupIdx] = key
	s.dedupIdx = (s.dedupIdx + 1) % dedupRingSize
	return false
}

func (s *FlashBlockSubscriber) processLog(log types.Log) {
	if len(log.Topics) == 0 {
		return
	}

	if s.isDuplicate(log) {
		return
	}

	nowMs := time.Now().UnixMilli()
	s.lastFlashTime.Store(nowMs)
	s.lastEventTime.Store(nowMs)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.latestState == nil {
		s.latestState = &poolState{
			SqrtPriceX96: new(uint256.Int),
			ReserveX:     new(uint256.Int),
			ReserveY:     new(uint256.Int),
		}
	}

	now := time.Now()

	switch log.Topics[0] {
	case topicStateUpdated:
		s.handleStateUpdated(log)
		s.latestState.StateUpdatedAt = now
	case topicSync:
		s.handleSync(log)
		s.latestState.ReservesUpdatedAt = now
	case topicSwapExecuted:
		// Replay against current cached pre-swap reserves; the matching Sync
		// log (later in tx order) will overwrite reserves on the next call.
		s.handleSwapExecuted(log)
	case topicConcentrationKSet:
		s.handleConcentrationKSet(log)
	case topicBlockDelaySet:
		s.handleBlockDelaySet(log)
	}

	s.latestState.BlockNumber = log.BlockNumber
}

func (s *FlashBlockSubscriber) handleStateUpdated(log types.Log) {
	values, err := coreABI.Events["StateUpdated"].Inputs.Unpack(log.Data)
	if err != nil {
		return
	}
	if len(values) < 3 {
		return
	}

	anchorBig, ok1 := values[0].(*big.Int)
	feeAskBig, ok2 := values[1].(*big.Int)
	feeBidBig, ok3 := values[2].(*big.Int)
	if !ok1 || !ok2 || !ok3 {
		return
	}

	s.latestState.SqrtPriceX96 = big256.FromBig(anchorBig)
	s.latestState.FeeAskX24 = uint32(feeAskBig.Uint64())
	s.latestState.FeeBidX24 = uint32(feeBidBig.Uint64())
	s.latestState.LatestUpdateBlock = log.BlockNumber
}

func (s *FlashBlockSubscriber) handleSync(log types.Log) {
	values, err := coreABI.Events["Sync"].Inputs.Unpack(log.Data)
	if err != nil {
		return
	}
	if len(values) < 2 {
		return
	}

	reserveX, ok1 := values[0].(*big.Int)
	reserveY, ok2 := values[1].(*big.Int)
	if !ok1 || !ok2 {
		return
	}

	s.latestState.ReserveX = big256.FromBig(reserveX)
	s.latestState.ReserveY = big256.FromBig(reserveY)
}

// handleSwapExecuted projects (dx, dy) onto cached reserves. The on-chain
// fix/incident contract does not mutate `anchorPrice` on swaps, so we do not
// touch SqrtPriceX96 here. The matching Sync log lands later in the same tx
// and overwrites reserves with the authoritative post-swap values.
func (s *FlashBlockSubscriber) handleSwapExecuted(log types.Log) {
	if s.latestState.ReserveX == nil || s.latestState.ReserveY == nil {
		return
	}

	values, err := coreABI.Events["SwapExecuted"].Inputs.Unpack(log.Data)
	if err != nil || len(values) < 5 {
		return
	}
	xToY, ok := values[1].(bool)
	if !ok {
		return
	}
	dxBig, ok1 := values[2].(*big.Int)
	dyBig, ok2 := values[3].(*big.Int)
	if !ok1 || !ok2 {
		return
	}

	dx, dy := big256.FromBig(dxBig), big256.FromBig(dyBig)
	if xToY {
		if s.latestState.ReserveY.Lt(dy) {
			return
		}
		s.latestState.ReserveX = new(uint256.Int).Add(s.latestState.ReserveX, dx)
		s.latestState.ReserveY = new(uint256.Int).Sub(s.latestState.ReserveY, dy)
	} else {
		if s.latestState.ReserveX.Lt(dx) {
			return
		}
		s.latestState.ReserveY = new(uint256.Int).Add(s.latestState.ReserveY, dy)
		s.latestState.ReserveX = new(uint256.Int).Sub(s.latestState.ReserveX, dx)
	}
}

func (s *FlashBlockSubscriber) handleConcentrationKSet(log types.Log) {
	values, err := coreABI.Events["ConcentrationKSet"].Inputs.Unpack(log.Data)
	if err != nil {
		return
	}
	if len(values) < 1 {
		return
	}

	k, ok := values[0].(uint32)
	if !ok {
		return
	}

	s.latestState.ConcentrationK = k
}

func (s *FlashBlockSubscriber) handleBlockDelaySet(log types.Log) {
	values, err := coreABI.Events["BlockDelaySet"].Inputs.Unpack(log.Data)
	if err != nil {
		return
	}
	if len(values) < 1 {
		return
	}

	bd, ok := values[0].(uint64)
	if !ok {
		return
	}

	s.latestState.BlockDelay = bd
}

func subscribeNewHeads(ctx context.Context, client *rpc.Client, ch chan<- *types.Header) (*rpc.ClientSubscription, error) {
	return client.EthSubscribe(ctx, ch, "newHeads")
}

func subscribeFilterLogs(
	ctx context.Context,
	client *rpc.Client,
	method string,
	addr common.Address,
	topic common.Hash,
	ch chan<- types.Log,
) (*rpc.ClientSubscription, error) {
	arg := map[string]interface{}{
		"address": addr,
		"topics":  []common.Hash{topic},
	}
	return client.Subscribe(ctx, "eth", ch, method, arg)
}

type logSub struct {
	ch  chan types.Log
	sub *rpc.ClientSubscription
}

func subscribePoolLogs(
	ctx context.Context,
	client *rpc.Client,
	addr common.Address,
	topics []struct {
		topic common.Hash
		name  string
	},
) ([]logSub, string, error) {
	for _, method := range []string{"pendingLogs", "logs"} {
		subs := make([]logSub, 0, len(topics))
		ok := true

		for _, t := range topics {
			ch := make(chan types.Log, 64)
			sub, err := subscribeFilterLogs(ctx, client, method, addr, t.topic, ch)
			if err != nil {
				for _, ls := range subs {
					ls.sub.Unsubscribe()
				}
				ok = false
				break
			}
			subs = append(subs, logSub{ch: ch, sub: sub})
		}

		if ok {
			return subs, method, nil
		}
	}

	return nil, "", fmt.Errorf("subscribe logs failed for all supported methods")
}
