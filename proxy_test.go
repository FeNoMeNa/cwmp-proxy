package cwmpproxy

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyHandler(t *testing.T) {
	cases := []struct {
		in, want io.Reader
	}{
		{
			bytes.NewReader([]byte(
				`
					<ParameterValueStruct>
						<Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name>
						<Value xsi:type="xsd:string">http://8.8.8.8:7547</Value>
					</ParameterValueStruct>
				`,
			)),

			bytes.NewReader([]byte(
				`
					<ParameterValueStruct>
						<Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name>
						<Value xsi:type="xsd:string">http://github.com/client?origin=http://8.8.8.8:7547</Value>
					</ParameterValueStruct>
				`,
			)),
		},

		{
			bytes.NewReader([]byte(`<EventStruct><EventCode>0 BOOTSTRAP</EventCode><CommandKey/></EventStruct>`)),
			bytes.NewReader([]byte(`<EventStruct><EventCode>0 BOOTSTRAP</EventCode><CommandKey/></EventStruct>`)),
		},

		{
			bytes.NewReader([]byte{}),
			bytes.NewReader([]byte{}),
		},
	}

	for _, c := range cases {
		server := fakeHttpServer(func(w http.ResponseWriter, r *http.Request) {
			compareReaders(t, c.want, r.Body)
		})

		p, _ := New(1234, server.URL)

		request, _ := http.NewRequest("GET", "http://github.com/", c.in)
		recorder := httptest.NewRecorder()
		p.handler().ServeHTTP(recorder, request)

		p.Close()
	}
}

func TestWakeupHandler(t *testing.T) {
	server := fakeHttpServer(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="wakeup-test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if username != "user" {
			t.Errorf("expected username: user")
			t.Errorf("     got username: %v", username)

		}

		if password != "pass" {
			t.Errorf("expected password: pass")
			t.Errorf("     got password: %v", password)
		}
	})

	cases := []struct {
		url  string
		code int
	}{
		{
			"http://github.com/client?origin=",
			http.StatusBadRequest,
		},

		{
			"http://github.com/client?origin=http://localhost:1342/",
			http.StatusBadRequest,
		},

		{
			"http://github.com/client?origin=" + server.URL,
			http.StatusOK,
		},
	}

	for _, c := range cases {
		request, _ := http.NewRequest("GET", c.url, nil)
		recorder := httptest.NewRecorder()

		wakeupHandler("user", "pass").ServeHTTP(recorder, request)

		if recorder.Code != c.code {
			t.Errorf("expected status code %v", c.code)
			t.Errorf("     got status code %v", recorder.Code)
		}
	}
}

func TestBasicAuthHandler(t *testing.T) {
	cases := []struct {
		username, password string
	}{
		{"user", "pass"},
		{"FeNoMeNa", "123456"},
	}

	for _, c := range cases {
		handler := func(username, password string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if username != c.username {
					t.Errorf("expected username: %v", c.username)
					t.Errorf("     got username: %v", username)

				}

				if password != c.password {
					t.Errorf("expected password: %v", c.password)
					t.Errorf("     got password: %v", password)
				}
			}
		}

		recorder := httptest.NewRecorder()

		request, _ := http.NewRequest("GET", "http://fake.com", nil)
		request.SetBasicAuth(c.username, c.password)

		basicAuthHandler(handler).ServeHTTP(recorder, request)
	}
}

func TestBasicAuthHandlerWithoutCredentials(t *testing.T) {
	handler := func(username, password string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

		}
	}

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://fake.com", nil)

	basicAuthHandler(handler).ServeHTTP(recorder, request)

	authHeader := recorder.Header().Get("WWW-Authenticate")

	if authHeader != `Basic realm="cwmp-proxy"` {
		t.Errorf("expected value of WWW-Authenticate: %v", `Basic realm="cwmp-proxy"`)
		t.Errorf("     got value of WWW-Authenticate: %v", authHeader)
	}

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status code: %v", http.StatusUnauthorized)
		t.Errorf("     got status code: %v", recorder.Code)
	}
}

func fakeHttpServer(callbacks ...func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, callback := range callbacks {
			callback(w, r)
		}
	})

	return httptest.NewServer(handler)
}

func compareReaders(t *testing.T, want io.Reader, got io.Reader) {
	gotBuffer, _ := ioutil.ReadAll(got)
	wantBuffer, _ := ioutil.ReadAll(want)

	if !bytes.Equal(gotBuffer, wantBuffer) {
		t.Errorf("expected %s", wantBuffer)
		t.Errorf("     got %s", gotBuffer)
	}
}
