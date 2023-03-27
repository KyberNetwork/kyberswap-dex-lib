package service

import (
	"github.com/DataDog/datadog-go/statsd"
)

func NewDogstatsd(host string) (*statsd.Client, error) {
	client, err := statsd.New(host + ":8125")
	if err != nil {
		return nil, err
	}

	client.Namespace = "kybernetwork."

	return client, nil
}
