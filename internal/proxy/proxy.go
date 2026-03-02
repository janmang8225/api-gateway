package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/janmang8225/api-gateway/internal/breaker"
)

const upstreamTimeout = 5 * time.Second

type Proxy struct {
	transport *http.Transport
}

func New() *Proxy {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	return &Proxy{
		transport: transport,
	}
}

func (p *Proxy) Forward(backendURL string, cb *breaker.CircuitBreaker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cb.Allow() {
			fmt.Printf("circuit breaker: blocking request to %s\n", backendURL)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("503 Service Unavailable — circuit open"))
			return
		}

		parsed, err := url.Parse(backendURL)
		if err != nil {
			cb.RecordFailure()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 Internal Server Error"))
			return
		}

		rp := httputil.NewSingleHostReverseProxy(parsed)
		rp.Transport = p.transport

		failed := false

		rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			failed = true
			cb.RecordFailure()

			if err == context.DeadlineExceeded {
				fmt.Println("proxy: upstream timeout")
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte("504 Gateway Timeout"))
				return
			}

			fmt.Printf("proxy error: %v\n", err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("502 Bad Gateway"))
		}

		ctx, cancel := context.WithTimeout(r.Context(), upstreamTimeout)
		defer cancel()

		r = r.WithContext(ctx)
		rp.ServeHTTP(w, r)

		if !failed {
			cb.RecordSuccess()
		}
	})
}