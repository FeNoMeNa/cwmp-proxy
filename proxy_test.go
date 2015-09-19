package main

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
			bytes.NewReader([]byte(`<ParameterValueStruct><Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name><Value xsi:type="xsd:string">http://8.8.8.8:7547</Value></ParameterValueStruct>`)),
			bytes.NewReader([]byte(`<ParameterValueStruct><Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name><Value xsi:type="xsd:string">http://github.com/client?origin=http://8.8.8.8:7547</Value></ParameterValueStruct>`)),
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
		server := fakeBackendServer(func(w http.ResponseWriter, r *http.Request) {
			compareReaders(t, c.want, r.Body)
		})

		p := NewProxy(1234, server.URL)

		request, _ := http.NewRequest("GET", "http://github.com/", c.in)
		recorder := httptest.NewRecorder()

		p.handler().ServeHTTP(recorder, request)
	}
}

func TestWakeupHandler(t *testing.T) {
	server := fakeBackendServer()

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

		wakeupHandler().ServeHTTP(recorder, request)

		if recorder.Code != c.code {
			t.Errorf("expected status code %v", c.code)
			t.Errorf("     got status code %v", recorder.Code)
		}
	}
}

func TestProxyAddressGenerating(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{8080, ":8080"},
		{9090, ":9090"},
	}

	for _, c := range cases {
		p := NewProxy(c.in, "http://fake.com/")
		got := p.address()

		if got != c.want {
			t.Errorf("expected %v", c.want)
			t.Errorf("     got %v", got)
		}
	}
}

func fakeBackendServer(callbacks ...func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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
