package main

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	cookiejar "github.com/juju/persistent-cookiejar"
)

type ProxyClientRotator struct {
	proxyHttpClients []*http.Client
	m                *sync.Mutex
	cookieJar        *cookiejar.Jar
	index            uint32
	useTimer         bool
	shuffle          bool
}

func NewProxyRotator(proxyUrls []string, cookieFile string, timeoutSec int, useTimer, shuffle bool) (*ProxyClientRotator, error) {
	var httpClients []*http.Client //Init array of http clients' pointer
	var cookieJar *cookiejar.Jar
	var err error

	if len(proxyUrls) <= 0 {
		return nil, errors.New("no proxies are given")
	}

	if cookieFile != "" {
		cookieJar, err = cookiejar.New(&cookiejar.Options{
			Filename: cookieFile,
		})
		if err != nil {
			return nil, err
		}
	}

	for _, url := range proxyUrls {
		client, err := createProxyClient(cookieJar, url, timeoutSec)
		if err != nil {
			return nil, err
		}
		httpClients = append(httpClients, client)
	}

	return &ProxyClientRotator{
		proxyHttpClients: httpClients,
		m:                &sync.Mutex{},
		cookieJar:        cookieJar,
		index:            0, //start from 0
		useTimer:         useTimer,
		shuffle:          shuffle,
	}, nil
}

func (r *ProxyClientRotator) GetProxyClient() (*http.Client, error) {
	var client *http.Client
	//Check if using cookie file
	if r.cookieJar != nil {
		//lock to save
		r.m.Lock()
		//Save the cookie data to cookie file
		defer r.cookieJar.Save()
		//Unlock
		r.m.Unlock()
	}

	currentIndex := r.index % uint32(len(r.proxyHttpClients))
	//if this is the new round (means index = 0) and shuffling is enabled
	if r.shuffle && currentIndex == 0 {
		r.proxyHttpClients = shuffleClients(r.proxyHttpClients)

	}
	client = r.proxyHttpClients[currentIndex]
	atomic.SwapUint32(&r.index, currentIndex+1)

	return client, nil
}
