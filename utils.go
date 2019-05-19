//Package goproxyrotator allows to create multiple proxy http clients
//as well as save cookie to persistent file
package goproxyrotator

import (
	"math/rand"
	"net/http"
	"net/url"
	"time"

	cookiejar "github.com/juju/persistent-cookiejar"
)

//createProxyClient create HTTP client that uses proxy
func createProxyClient(cookieJar *cookiejar.Jar, proxyURL string, timeoutSec int) (*http.Client, error) {
	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxyURL),
	}
	timeout := time.Duration(timeoutSec) * time.Second
	if cookieJar != nil {
		return &http.Client{
			Transport: transport,
			Timeout:   timeout,
			Jar:       cookieJar,
		}, nil
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

//createCookieJar create cookie jar based on given file path
func createCookieJar(fileName string) (*cookiejar.Jar, error) {
	if fileName == "" {
		return nil, nil
	}
	cookieJar, err := cookiejar.New(&cookiejar.Options{
		Filename: fileName,
	})
	if err != nil {
		return nil, err
	}
	return cookieJar, nil
}

//shuffleClients shuffle enhanced proxy client
func shuffleClients(clients []*EnhancedProxyClient) []*EnhancedProxyClient {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]*EnhancedProxyClient, len(clients))
	perm := r.Perm(len(clients))
	for i, randIndex := range perm {
		ret[i] = clients[randIndex]
	}
	return ret
}

//isClientUsable is a function to check if a http client is usable.
//There will be some bad cases that some proxies are not online
func isClientUsable(client *http.Client) bool {
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		return false
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	return true
}
