package httpclient

import (
	"net/http"
	"time"
)

const (
	defaultTimeout       = 5 * time.Second
	defaultMaxConnection = 50
)

type options struct {
	maxConnection int
	timeout       time.Duration
}

// Option sets options for http client.
type Option func(*options)

func NewStdHTTPClient(opts ...Option) *http.Client {
	o := options{
		maxConnection: defaultMaxConnection,
		timeout:       defaultTimeout,
	}

	for _, opt := range opts {
		opt(&o)
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = o.maxConnection
	t.MaxConnsPerHost = o.maxConnection
	t.MaxIdleConnsPerHost = o.maxConnection

	return &http.Client{
		Timeout:   o.timeout,
		Transport: t,
	}
}
