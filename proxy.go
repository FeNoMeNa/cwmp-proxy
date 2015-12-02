package cwmpproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/FeNoMeNa/goha"
)

// Proxy represents an CWMP proxy server. There may be multiple backend endpoints that will accept
// the incoming requests.
type Proxy struct {
	listener net.Listener
	backend  *url.URL
}

// NewProxy creates and initializes a new CWMP proxy. If the desired port is not free an error
// will be returned.
func New(port int, backend string) (*Proxy, error) {
	u, err := url.Parse(backend)

	if err != nil {
		return new(Proxy), err
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		return new(Proxy), err
	}

	return &Proxy{l, u}, nil
}

// Start starts the CWMP proxy. It registers two main http handlers, the first one is the proxy
// handler, the second one is related with the CPE waking up.
func (p *Proxy) Start() error {
	http.Handle("/", p.handler())
	http.Handle("/client", basicAuthHandler(wakeupHandler))

	return http.Serve(p.listener, nil)
}

// Close terminates the CWMP proxy. It simply closes the TCP server listener.
func (p *Proxy) Close() error {
	return p.listener.Close()
}

// handler is the core of the CWMP proxy. It uses the internal ReverseProxy to implement
// the proxy logic that will send the incoming requests the backend servers. We should
// notice that the handler modifies the received CWMP content. If the request contains a
// connection url it will be replaced with a custom defined.
func (p *Proxy) handler() http.Handler {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			cwmp := newCwmpMessage(req)
			cwmp.replaceConnectionUrl(req.Host)

			req.URL.Scheme = p.backend.Scheme
			req.URL.Host = p.backend.Host
		},
	}
}

// wakeupHandler wakes up a concrete CPE. When the wakeupHandler endpoint is called with
// a valid connection url, an authorized GET request is send to that url.
func wakeupHandler(username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.FormValue("origin")

		if origin == "" {
			http.Error(w, "The origin connection URL should be provided!", http.StatusBadRequest)
			return
		}

		c := goha.NewClient(username, password)
		resp, err := c.Get(origin)

		if err != nil {
			http.Error(w, "An error occurred with the CPE communication!", http.StatusBadRequest)
			return
		}

		w.WriteHeader(resp.StatusCode)
	}
}

// basicAuthHandler is a handler wrapper that will obtain the username and password encoded
// with basic access authentication scheme. If the Authorization header is not provided the
// wrapper will respond with 401 status code.
func basicAuthHandler(handler func(string, string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="cwmp-proxy"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler(username, password).ServeHTTP(w, r)
	}
}
