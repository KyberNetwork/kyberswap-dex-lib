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

func GetClient() *statsd.Client {
	return client
}

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

	return client, nil
}
