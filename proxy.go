package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	p := NewProxy(8080, "http://github.com/")
	p.Start()
}

type Proxy struct {
	port    int
	backend *url.URL
}

func NewProxy(port int, backend string) *Proxy {
	u, _ := url.Parse(backend)

	return &Proxy{port, u}
}

func (p *Proxy) Start() {
	http.Handle("/", p.handler())
	http.Handle("/client", wakeupHandler())

	http.ListenAndServe(p.address(), nil)
}

func (p *Proxy) handler() http.Handler {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			cwmp := newCwmpMessage(req.Host)
			cwmp.replaceConnectionUrl(req)

			req.URL.Scheme = p.backend.Scheme
			req.URL.Host = p.backend.Host
		},
	}
}

func wakeupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.FormValue("origin")

		if origin == "" {
			http.Error(w, "The origin connection URL should be provided!", http.StatusBadRequest)
			return
		}

		_, err := http.Get(origin)

		if err != nil {
			http.Error(w, "An error occurred with the CPE communication!", http.StatusBadRequest)
			return
		}
	}
}

func (p *Proxy) address() string {
	return fmt.Sprintf(":%d", p.port)
}
