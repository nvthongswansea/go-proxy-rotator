//Package goProxyRotator allows to create multiple proxy http clients
//as well as save cookie to persistent file
package goProxyRotator

import (
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	cookiejar "github.com/juju/persistent-cookiejar"
)

//EnhancedProxyClient a struct that contain the
//http client, proxy infomation, cookie jar
type EnhancedProxyClient struct {
	HTTPclient *http.Client
	proxyURL   string
	m          *sync.Mutex
	cookie     *cookiejar.Jar
}

//GetHTTPclient get http client from the enhanced proxy client
func (c *EnhancedProxyClient) GetHTTPclient() *http.Client {
	return c.HTTPclient
}

//SaveCookie save cookie to file
func (c *EnhancedProxyClient) SaveCookie() {
	//Check if using cookie file
	if c.cookie != nil {
		//lock to save
		c.m.Lock()
		//Save the cookie data to cookie file
		c.cookie.Save()
		//Unlock
		c.m.Unlock()
	}
}

//ProxyClientRotator is an slice of http clients
type ProxyClientRotator struct {
	proxyHTTPClients []*EnhancedProxyClient
	cookieJars       map[string]*cookiejar.Jar
	createdAt        time.Time
	delayedTimeMsc   int64
	index            uint32
	shuffle          bool
}

//NewProxyRotator create new proxy client rotator.
//Proxy url has to be in form: 'protocol://username:password@proxy_address:port'
//The number of proxies' URLs has to be equal to the number of cookie files.
//e.g. proxyURLs = ['https://username:password@1.2.3.4:1080', 'https://username:password@2.3.4.5:1080',
//'https://username:password@1.1.1.1:1080', 'https://username:password@2.2.2.2:1080']
//cookieFiles = ['', 'proxy1', 'proxy2', 'proxy1'].
//If cookie file is '', the corresponding proxy client does not use persistent cookie.
//Many proxy clients can use the same cookie file.
//If delayed time is set, the next client got from rotator is based on time interval.
//e.g. delayedTimeMsc = 1000, the next client will be fired every 1 second.
//If call GetProxyClient() many times within 1 sec, the same client will be returned.
func NewProxyRotator(proxyURLs []string, cookieFiles []string, timeoutSec int, delayedTimeMsc int64, shuffle bool) (*ProxyClientRotator, error) {
	var httpClientsWithProxies []*EnhancedProxyClient //Init array of http clients' pointer
	var cookieJars = make(map[string]*cookiejar.Jar)
	var warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	if len(proxyURLs) <= 0 {
		return nil, errors.New("no proxies are given")
	}

	for _, cookieFile := range cookieFiles {
		//If the cookie jar has not created, create one and at to cookieJars map
		if _, ok := cookieJars[cookieFile]; !ok {
			cookieJar, err := createCookieJar(cookieFile)
			if err != nil {
				return nil, err
			}
			cookieJars[cookieFile] = cookieJar
		}
	}

	for i, URL := range proxyURLs {
	COOKIEFILESLOOP:
		for j, cookieFile := range cookieFiles {
			if i == j {
				client, err := createProxyClient(cookieJars[cookieFile], URL, timeoutSec) //clients that use same cookie file will use same cookie jar
				if err != nil {
					return nil, err
				}
				//Check if the proxy is alive
				if isUsable := isClientUsable(client); isUsable {
					httpClientsWithProxies = append(httpClientsWithProxies, &EnhancedProxyClient{
						HTTPclient: client,
						proxyURL:   URL,
						m:          &sync.Mutex{},
					})
				}
				warning.Println(URL, "is removed as it is not usable. Please check your proxy.")
				break COOKIEFILESLOOP
			}
		}
	}

	return &ProxyClientRotator{
		proxyHTTPClients: httpClientsWithProxies,
		cookieJars:       cookieJars,
		createdAt:        time.Now(),
		index:            0, //start from 0
		delayedTimeMsc:   delayedTimeMsc,
		shuffle:          shuffle,
	}, nil
}

//AddProxyClient add a new proxy client to the rotator
func (r *ProxyClientRotator) AddProxyClient(proxyURL, cookieFile string, timeoutSec int) error {
	var currentClients = r.proxyHTTPClients
	var cookieJar *cookiejar.Jar
	var err error
	//Check if the cookie file is loaded in rotator
	if v, ok := r.cookieJars[cookieFile]; ok {
		//use the cookie jar that has already created
		cookieJar = v
	} else {
		cookieJar, err = createCookieJar(cookieFile)
		if err != nil {
			return err
		}
	}

	newClient, err := createProxyClient(cookieJar, proxyURL, timeoutSec)
	if err != nil {
		return err
	}
	if isUsable := isClientUsable(newClient); isUsable {
		currentClients = append(currentClients, &EnhancedProxyClient{
			HTTPclient: newClient,
			proxyURL:   proxyURL,
		})
	}
	r.proxyHTTPClients = currentClients
	return nil
}

//GetProxyClient get an enhanced proxy client from rotator
func (r *ProxyClientRotator) GetProxyClient() *EnhancedProxyClient {
	var currentIndex int

	//if use timer instead of using index
	if r.delayedTimeMsc != 0 {
		//get the interval from created time to current time in milisecs
		interval := time.Now().Sub(r.createdAt).Nanoseconds() / 1e6
		if interval != 0 {
			currentIndex = int((interval / r.delayedTimeMsc) % int64(len(r.proxyHTTPClients)))
		}
	} else {
		currentIndex := r.index % uint32(len(r.proxyHTTPClients))
		atomic.SwapUint32(&r.index, currentIndex+1)
	}

	//if this is the new round (means index = 0) and shuffling is enabled
	if r.shuffle && currentIndex == 0 {
		r.proxyHTTPClients = shuffleClients(r.proxyHTTPClients)
	}

	return r.proxyHTTPClients[currentIndex]
}

//CheckHealthAll checks health of all http clients
//return a map (key - URL of proxy, value - true or false)
func (r *ProxyClientRotator) CheckHealthAll() map[string]bool {
	result := make(map[string]bool)
	for _, enhancedProxyClient := range r.proxyHTTPClients {
		isUsable := isClientUsable(enhancedProxyClient.HTTPclient)
		result[enhancedProxyClient.proxyURL] = isUsable
	}
	return result
}
