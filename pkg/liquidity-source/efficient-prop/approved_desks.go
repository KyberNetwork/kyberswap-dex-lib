package efficientprop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

const (
	// defaultDeskListMaxAge is how long a fetched desk list rules where Config.ApprovedDesksMaxAge is unset.
	defaultDeskListMaxAge = 10 * time.Minute
	// deskListEvery is how often the list is fetched at most, so how long a desk the list dropped is still quoted at
	// most; deskListTimeout bounds one fetch.
	deskListEvery   = 15 * time.Second
	deskListTimeout = 3 * time.Second
	// maxDeskListBytes and maxDeskListDesks bound a list: a longer answer is refused whole.
	maxDeskListBytes = 64 << 10
	maxDeskListDesks = 1024
)

// deskListAnswer is the list a desk list url answers: the chain it is for, when it was last changed (unix seconds)
// and the desk contracts approved on that chain.
type deskListAnswer struct {
	ChainID   uint64   `json:"chainId"`
	UpdatedAt uint64   `json:"updatedAt"`
	Desks     []string `json:"desks"`
}

// approvedDesks is the maker gate: the desk contracts whose listings are listed and quoted. A listing's maker is its
// desk, and anyone may open a desk and a listing on the front, so no listing is quoted for being there: only where
// its desk's contract is approved.
//
// The static set is Config.ApprovedDesks. Where Config.ApprovedDesksURL is set, its list is fetched every
// deskListEvery at most, and the last list accepted rules while its fetch is at most maxAge old; once it is older (the
// url failing, or answering a list that is refused) the static set rules, less every desk that list left out. So a
// failing url never brings a desk in: not one of the list's that the static set does not name, and not one of the
// static set's that the list had dropped. A list is refused whole where its status is not 200, its body is above
// maxDeskListBytes, it names another chain, an address that is not one, more than maxDeskListDesks desks, no
// updatedAt, or an updatedAt before the held list's (an older list served again).
type approvedDesks struct {
	static  map[common.Address]struct{}
	url     string
	chainID uint64
	maxAge  time.Duration
	client  *http.Client
	now     func() time.Time
	warn    func(err error)

	mu        sync.Mutex
	listed    map[common.Address]struct{} // the last list accepted; nil where none was
	fetchedAt time.Time
	updatedAt uint64
	nextAt    time.Time
}

func newApprovedDesks(cfg *Config) *approvedDesks {
	d := &approvedDesks{static: make(map[common.Address]struct{}, len(cfg.ApprovedDesks)), url: cfg.ApprovedDesksURL,
		chainID: uint64(cfg.ChainID), maxAge: defaultDeskListMaxAge, now: time.Now,
		client: &http.Client{Timeout: deskListTimeout, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL.Scheme != "https" {
				return errors.New("redirected off https")
			}
			return nil
		}}}
	for _, a := range cfg.ApprovedDesks {
		d.static[common.HexToAddress(a)] = struct{}{}
	}
	if cfg.ApprovedDesksMaxAge != 0 {
		d.maxAge = time.Duration(cfg.ApprovedDesksMaxAge) * time.Second
	}
	d.warn = func(err error) {
		logger.WithFields(logger.Fields{"dexId": cfg.dexID(), "url": shownURL(d.url), "error": err.Error()}).
			Warn("efficient-prop: the approved desks list was not taken")
	}
	if len(d.static) == 0 && d.url == "" {
		logger.WithFields(logger.Fields{"dexId": cfg.dexID()}).
			Warn("efficient-prop: no approved desks are configured: no listing is listed or quoted")
	}
	return d
}

// approved is whether desk, a desk contract, is approved now.
func (d *approvedDesks) approved(ctx context.Context, desk common.Address) bool {
	if desk == (common.Address{}) {
		return false
	}
	d.refresh(ctx)
	d.mu.Lock()
	defer d.mu.Unlock()
	_, static := d.static[desk]
	if d.listed == nil {
		return static
	}
	_, listed := d.listed[desk]
	if d.now().Sub(d.fetchedAt) <= d.maxAge {
		return listed
	}
	return static && listed
}

// refresh fetches the list where one is due. One caller fetches at a time, and the others go on with the list held.
func (d *approvedDesks) refresh(ctx context.Context) {
	if d.url == "" {
		return
	}
	d.mu.Lock()
	now := d.now()
	if now.Before(d.nextAt) {
		d.mu.Unlock()
		return
	}
	d.nextAt = now.Add(deskListEvery)
	held := d.updatedAt
	d.mu.Unlock()

	a, err := d.fetch(ctx)
	if err == nil && a.UpdatedAt < held {
		err = fmt.Errorf("its updatedAt %d is before the held list's %d", a.UpdatedAt, held)
	}
	if err != nil {
		if d.warn != nil {
			d.warn(shownError(err))
		}
		return
	}
	listed := make(map[common.Address]struct{}, len(a.Desks))
	for _, s := range a.Desks {
		listed[common.HexToAddress(s)] = struct{}{}
	}
	d.mu.Lock()
	if a.UpdatedAt >= d.updatedAt {
		d.listed, d.updatedAt, d.fetchedAt = listed, a.UpdatedAt, d.now()
	}
	d.mu.Unlock()
}

// fetch is the url's list, checked.
func (d *approvedDesks) fetch(ctx context.Context) (a deskListAnswer, err error) {
	u, err := neturl.Parse(d.url)
	if err != nil || u.Scheme != "https" {
		return a, errors.New("the url is not https")
	}
	ctx, cancel := context.WithTimeout(ctx, deskListTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, nil)
	if err != nil {
		return a, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return a, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return a, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDeskListBytes+1))
	if err != nil {
		return a, err
	}
	if len(body) > maxDeskListBytes {
		return a, fmt.Errorf("the list is above %d bytes", maxDeskListBytes)
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return a, err
	}
	if a.ChainID != d.chainID {
		return a, fmt.Errorf("the list is chain %d's, not chain %d's", a.ChainID, d.chainID)
	}
	if a.UpdatedAt == 0 {
		return a, errors.New("the list names no updatedAt")
	}
	if len(a.Desks) > maxDeskListDesks {
		return a, fmt.Errorf("the list names %d desks, above %d", len(a.Desks), maxDeskListDesks)
	}
	for _, s := range a.Desks {
		if !common.IsHexAddress(s) || common.HexToAddress(s) == (common.Address{}) {
			return a, fmt.Errorf("the list names %q, which is no address", s)
		}
	}
	return a, nil
}
