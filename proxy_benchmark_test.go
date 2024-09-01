//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//

package cbreaker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/proxy"
)

func BenchmarkNewCircuitBreakerMiddleware_ok(b *testing.B) {
	p := NewMiddleware(&cfg)(dummyProxy(&proxy.Response{}, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

func BenchmarkNewCircuitBreakerMiddleware_ko(b *testing.B) {
	p := NewMiddleware(&cfg)(dummyProxy(nil, errors.New("sample error")))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

func BenchmarkNewCircuitBreakerMiddleware_burst(b *testing.B) {
	err := errors.New("sample error")
	p := NewMiddleware(&cfg)(burstProxy(&proxy.Response{}, err, 100, 6))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

var cfg = config.Backend{
	ExtraConfig: map[string]interface{}{
		Namespace: map[string]interface{}{
			"command_name":            "test_cmd",
			"timeout":                 100.0,
			"max_concurrent_requests": 100.0,
			"error_percent_threshold": 1.0,
		},
	},
}

func dummyProxy(r *proxy.Response, err error) proxy.Proxy {
	return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		return r, err
	}
}

func burstProxy(r *proxy.Response, err error, ok, ko int) proxy.Proxy {
	tmp := make([]bool, ok+ko)
	for i := 0; i < ok+ko; i++ {
		tmp[i] = i < ok
	}
	calls := uint64(0)
	return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		total := atomic.AddUint64(&calls, 1) - 1
		if tmp[total%uint64(len(tmp))] {
			return r, nil
		}
		return nil, err
	}
}
