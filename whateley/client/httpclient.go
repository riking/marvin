package client

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Headers    http.Header
	httpClient http.Client
	options    Options
}

type Options struct {
	CacheDir  string
	UserAgent string
	Headers   http.Header
}

type printingRoundTripper struct {
	parent http.RoundTripper
}

func (p *printingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if p.parent == nil {
		p.parent = http.DefaultTransport
	}

	fmt.Printf("> %s %s\n", req.Method, req.URL.String())
	resp, err := p.parent.RoundTrip(req)
	fmt.Printf("< %s %s\n", req.Method, req.URL.String())
	return resp, err
}

func New(opts Options) *Client {
	c := new(Client)
	if opts.UserAgent == "" {
		opts.UserAgent = "Client Name Not Set (+github.com/riking/whateley)"
	}
	c.Headers = opts.Headers
	if c.Headers == nil {
		c.Headers = make(http.Header)
	}
	c.Headers.Set("User-Agent", opts.UserAgent)
	c.httpClient.Jar, _ = cookiejar.New(nil)
	c.httpClient.Timeout = 15 * time.Second
	c.httpClient.Transport = &printingRoundTripper{c.httpClient.Transport}

	if opts.CacheDir != "" {
		cache := diskcache.New(opts.CacheDir)
		tr := httpcache.NewTransport(cache)
		tr.Transport = c.httpClient.Transport
		c.httpClient.Transport = tr
	}
	return c
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for k := range c.Headers {
		req.Header.Set(k, c.Headers.Get(k))
	}
	return c.httpClient.Do(req)
}

func (c *Client) Document(req *http.Request) (*goquery.Document, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Header.Write(os.Stdout)
	return goquery.NewDocumentFromResponse(resp)
}
