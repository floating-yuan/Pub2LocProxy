package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func sendToNatGwServer() {
	// Send GET request
	addr := urlForNatGwServer
	if addr == "" {
		addr = DefaultUrlForNatGwServer
	}

	reader := strings.NewReader("request body....")
	readCloser := ioutil.NopCloser(reader)

	u, err := url.Parse(addr)
	if err != nil {
		log.Println("url.Parse(addr) failed: ", err)
		return
	}
	u.Path = "/aaa/bb/ddd"

	resp, err := http.DefaultClient.Do(&http.Request{
		Header: http.Header{
			"X-Request-Custom-Header-A": []string{"Req-A-Value"},
			"X-Request-Custom-Header-B": []string{"Req-B-Value"},
		},
		Body: readCloser,
		URL:  u,
	})

	if err != nil {
		log.Fatal("Request failed: ", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	log.Println("ioutil.ReadAll(resp.Body) begin")
	body, err := ioutil.ReadAll(resp.Body)
	log.Println("ioutil.ReadAll(resp.Body) end")
	if err != nil {
		log.Fatal("Failed to read response body: ", err)
	}

	// Print the response
	fmt.Println("Response headers:", resp.Header)
	fmt.Println("Response body:", string(body))
	fmt.Println()
}
