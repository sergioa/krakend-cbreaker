//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//

package cbreaker

import (
	"context"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/proxy"
)

// BackendFactory adds a cb middleware wrapping the internal factory
func BackendFactory(next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return NewMiddleware(cfg)(next(cfg))
	}
}

// NewMiddleware builds a middleware based on the extra config params or fallbacks to the next proxy
func NewMiddleware(remote *config.Backend) proxy.Middleware {
	data := ConfigGetter(remote.ExtraConfig).(Config)
	if data == ZeroCfg {
		return proxy.EmptyMiddleware
	}

	cb := NewCommand(data)

	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return NewCircuitBreakerRequest(cb, next[0])
	}
}

func NewCircuitBreakerRequest(cb *HystrixCommand, next proxy.Proxy) proxy.Proxy {
	var response *proxy.Response
	return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
		err := cb.Execute(func() error {
			var err error
			response, err = next(ctx, request)
			return err
		}, nil)
		return response, err
	}
}
