package main

import (
	"math/rand"
	"net/http"
	"net/url"
	"time"

	cookiejar "github.com/juju/persistent-cookiejar"
)

func createProxyClient(cookieJar *cookiejar.Jar, proxyURL string, timeoutSec int) (*http.Client, error) {
	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	if cookieJar != nil {
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
			},
			Timeout: time.Duration(timeoutSec) * time.Second,
			Jar:     cookieJar,
		}, nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(parsedProxyURL),
		},
		Timeout: time.Duration(timeoutSec) * time.Second,
	}, nil
}

func shuffleClients(clients []*http.Client) []*http.Client {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]*http.Client, len(clients))
	perm := r.Perm(len(clients))
	for i, randIndex := range perm {
		ret[i] = clients[randIndex]
	}
	return ret
}
