package cdnproxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const cdnIntra = "cdn.intra.42.fr"

type customDNSCache struct {
	mu      sync.Mutex
	res     []string
	expires time.Time
}

func (c *customDNSCache) Get() (addrs []string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.expires.Before(time.Now()) || c.res == nil {
		res, err := net.LookupHost(cdnIntra)
		if err != nil {
			return nil, err
		}
		c.res = res
		c.expires = time.Now().Add(30 * time.Minute)
		return c.res, nil
	}
	return c.res, nil
}

var cdnDNS customDNSCache
var defaultDialer net.Dialer

func ipOnlyDialer(resolvedAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if host == cdnIntra {
			addr = net.JoinHostPort(resolvedAddr, port)
		}
		return defaultDialer.DialContext(ctx, network, addr)
	}
}

type transportList struct {
	mu sync.Mutex
	m  map[string]*http.Transport
}

func (t *transportList) New(resolvedAddr string) *http.Client {
	t.mu.Lock()
	defer t.mu.Unlock()

	tr, ok := t.m[resolvedAddr]
	if ok {
		return &http.Client{Transport: tr}
	}
	tr = &http.Transport{
		DialContext: ipOnlyDialer(resolvedAddr),
	}
	t.m[resolvedAddr] = tr
	return &http.Client{Transport: tr}
}

var clientCache transportList

// Take the request, and fan-out to every intra CDN host available.
// Respond with the first non-error response offered, or a 404 if all hosts agree on a
// 404, or all errors otherwise.
func ProxyIntraCDN(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "ETag, Last-Modified")
	w.Header().Set("Access-Control-Max-Age", "36000")
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}

	pu, _ := url.ParseRequestURI(r.URL.RequestURI())
	pu.Scheme = "https"
	pu.Host = "cdn.intra.42.fr"
	pReq, err := http.NewRequest(r.Method, pu.String(), nil)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err)
		return
	}
	if r.Header.Get("Range") != "" {
		pReq.Header.Set("Range", r.Header.Get("Range"))
	}
	addrs, err := cdnDNS.Get()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err)
		return
	}
	var wg sync.WaitGroup
	var respCh = make(chan *http.Response, len(addrs))
	var errCh = make(chan error, len(addrs))
	wg.Add(len(addrs))
	for i := range addrs {
		go func(i int) {
			defer wg.Done()
			cl := clientCache.New(addrs[i])
			req := pReq.WithContext(r.Context())
			resp, err := cl.Do(req)
			if err != nil {
				errCh <- errors.Wrapf(err, "proxy %s", addrs[i])
			} else {
				respCh <- resp
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(respCh)
	}()

	any404 := false
	gotResponse := false

	var bodyDoneWg sync.WaitGroup
	for resp := range respCh {
		if resp == nil {
			continue
		}
		if gotResponse {
			// don't need you
			resp.Body.Close()
			continue
		}

		if resp.StatusCode == 200 || resp.StatusCode == 206 {
			// GOT IT
			gotResponse = true
			for k, v := range resp.Header {
				if k == "Server" || k == "Connection" || k == "Access-Control-Allow-Origin:" {
					continue
				}
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			bodyDoneWg.Add(1)
			go func() {
				io.Copy(w, resp.Body)
				resp.Body.Close()
				bodyDoneWg.Done()
			}()
			continue
		} else if resp.StatusCode == 404 {
			any404 = true
		} else {
			errCh <- fmt.Errorf("proxy: backing server returned unexpected status code %d", resp.StatusCode)
		}
		resp.Body.Close()
	}
	close(errCh)

	if !gotResponse {
		// Failure case
		if any404 {
			http.NotFound(w, r)
			return
		} else {
			w.WriteHeader(502)
			for v := range errCh {
				fmt.Fprintln(w, v)
			}
			return
		}
	}

	bodyDoneWg.Wait()
}
