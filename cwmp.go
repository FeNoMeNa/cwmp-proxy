package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type cwmpMessage struct {
	host string
}

func newCwmpMessage(host string) *cwmpMessage {
	return &cwmpMessage{host}
}

func (i *cwmpMessage) replaceConnectionUrl(request *http.Request) {
	content, err := ioutil.ReadAll(request.Body)

	if err != nil {
		return
	}

	message := string(content)

	url, err := getConnectionUrl(message)

	if err == nil {
		newUrl := fmt.Sprintf("http://%s/client?origin=%s", i.host, url)
		message = strings.Replace(message, url, newUrl, 1)
	}

	length := len(message)
	buffer := bytes.NewReader([]byte(message))

	request.Body = ioutil.NopCloser(buffer)
	request.ContentLength = int64(length)
}

func getConnectionUrl(message string) (string, error) {
	index := strings.Index(message, "ConnectionRequestURL")

	if index == -1 {
		return "", fmt.Errorf("The ConnectionRequestURL is not found!")
	}

	start := index + strings.Index(message[index:], "http")
	end := start + strings.Index(message[start:], "<")

	return message[start:end], nil
}
