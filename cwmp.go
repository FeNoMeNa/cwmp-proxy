package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type cwmpMessage struct {
	req *http.Request
}

func newCwmpMessage(req *http.Request) *cwmpMessage {
	return &cwmpMessage{req}
}

func (i *cwmpMessage) replaceConnectionUrl(host string) {
	content, err := ioutil.ReadAll(i.req.Body)

	if err != nil {
		return
	}

	message := string(content)

	url, ok := getConnectionUrl(message)

	if ok {
		newUrl := fmt.Sprintf("http://%s/client?origin=%s", host, url)
		message = strings.Replace(message, url, newUrl, 1)
	}

	length := len(message)
	buffer := bytes.NewReader([]byte(message))

	i.req.Body = ioutil.NopCloser(buffer)
	i.req.ContentLength = int64(length)
}

func getConnectionUrl(message string) (string, bool) {
	index := strings.Index(message, "ConnectionRequestURL")

	if index == -1 {
		return "", false
	}

	start := index + strings.Index(message[index:], "http")
	end := start + strings.Index(message[start:], "<")

	return message[start:end], true
}
