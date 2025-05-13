// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

// Option is a functional option for customizing static signatures.
type Option func(*options)

type options struct {
	UserAgent  string
	RetryCount uint
	Logger     interface{}

	// Client-side rate limiting to avoid rekor 429s.
	// This is the only real difference from upstream.
	// I'd rather just make the transport pluggable if we upstream this.
	limiter              *rate.Limiter
	defaultHttpTransport http.RoundTripper
}

const (
	// DefaultRetryCount is the default number of retries.
	DefaultRetryCount = 3
)

func makeOptions(opts ...Option) *options {
	o := &options{
		UserAgent:  "",
		RetryCount: DefaultRetryCount,
		// A little bird told me that rekor allows 500 requests per minute.
		// We want to stay well under that, so we'll round down to 5 QPS.
		limiter:              rate.NewLimiter(5.0, 1),
		defaultHttpTransport: cleanhttp.DefaultPooledTransport(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// WithUserAgent sets the media type of the signature.
func WithUserAgent(userAgent string) Option {
	return func(o *options) {
		o.UserAgent = userAgent
	}
}

// WithRetryCount sets the number of retries.
func WithRetryCount(retryCount uint) Option {
	return func(o *options) {
		o.RetryCount = retryCount
	}
}

// WithLogger sets the logger; it must implement either retryablehttp.Logger or retryablehttp.LeveledLogger; if not, this will not take effect.
func WithLogger(logger interface{}) Option {
	return func(o *options) {
		switch logger.(type) {
		case retryablehttp.Logger, retryablehttp.LeveledLogger:
			o.Logger = logger
		}
	}
}

type roundTripper struct {
	http.RoundTripper
	UserAgent string
	limiter   *rate.Limiter
}

// RoundTrip implements `http.RoundTripper`
func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", rt.UserAgent)

	// Blocks to avoid hitting rate limits.
	if err := rt.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("waiting for rate limiter: %w", err)
	}

	return rt.RoundTripper.RoundTrip(req)
}

func createRoundTripper(o *options) http.RoundTripper {
	return &roundTripper{
		RoundTripper: o.defaultHttpTransport,
		UserAgent:    o.UserAgent,
		limiter:      o.limiter,
	}
}
