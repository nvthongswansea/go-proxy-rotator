package main

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	cookiejar "github.com/juju/persistent-cookiejar"
)

type ProxyClientRotator struct {
	proxyHTTPClients []*http.Client
	m                *sync.Mutex
	cookieJar        *cookiejar.Jar
	createdAt        time.Time
	delayedTimeMsc   int64
	index            uint32
	shuffle          bool
}

func NewProxyRotator(proxyURLs []string, cookieFile string, timeoutSec int, delayedTimeMsc int64, shuffle bool) (*ProxyClientRotator, error) {
	var httpClients []*http.Client //Init array of http clients' pointer
	var cookieJar *cookiejar.Jar
	var err error

	if len(proxyURLs) <= 0 {
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

	for _, URL := range proxyURLs {
		client, err := createProxyClient(cookieJar, URL, timeoutSec)
		if err != nil {
			return nil, err
		}
		httpClients = append(httpClients, client)
	}

	return &ProxyClientRotator{
		proxyHTTPClients: httpClients,
		m:                &sync.Mutex{},
		cookieJar:        cookieJar,
		createdAt:        time.Now(),
		index:            0, //start from 0
		delayedTimeMsc:   delayedTimeMsc,
		shuffle:          shuffle,
	}, nil
}

func (r *ProxyClientRotator) GetProxyClient() (*http.Client, error) {
	var client *http.Client
	var currentIndex int
	//Check if using cookie file
	if r.cookieJar != nil {
		//lock to save
		r.m.Lock()
		//Save the cookie data to cookie file
		defer r.cookieJar.Save()
		//Unlock
		r.m.Unlock()
	}

	//if use timer instead of using index
	if r.delayedTimeMsc != 0 {
		interval := time.Now().Sub(r.createdAt).Nanoseconds() / 1e6
		if interval != 0 {
			currentIndex = int((interval / r.delayedTimeMsc) % int64(len(r.proxyHTTPClients)))
		}
	} else {
		currentIndex := r.index % uint32(len(r.proxyHTTPClients))
		//if this is the new round (means index = 0) and shuffling is enabled
		if r.shuffle && currentIndex == 0 {
			r.proxyHTTPClients = shuffleClients(r.proxyHTTPClients)

		}
		atomic.SwapUint32(&r.index, currentIndex+1)
	}

	client = r.proxyHTTPClients[currentIndex]
	return client, nil
}
