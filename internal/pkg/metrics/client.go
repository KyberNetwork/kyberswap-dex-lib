package metrics

import (
	"fmt"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
)

var (
	once sync.Once

	client *statsd.Client
)

// For now, we support both VanPT & DataDog for backward compatibility.
// When DataDog deprecates, we can simply disable it through env.
// TODO: Deprecate related DataDog code after VanPT runs stably.
func InitClient(config Config) (*statsd.Client, error) {
	if client != nil {
		return client, nil
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	statClient, err := statsd.New(addr, statsd.WithNamespace(config.Namespace))
	if err != nil {
		return nil, err
	}

	once.Do(func() {
		client = statClient
	})

	// VanPT client is init through import in main.

	return client, nil
}
