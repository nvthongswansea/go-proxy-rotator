# go-proxy-rotator
**go-proxy-rotator** is a Golang library that let you to manage a group of proxy clients. The way it works is similar to an anaconda pistol (you can google it to see the pistol's photos) works. Every time the pistol is triggered, one different bullet will be fired.
**Usage:**
```go
    proxies := []string{
		"http://173.249.43.105:3128",
		"socks5://18.208.234.52:1080",
	}
	cookieFiles := []string{
		"",
		"testcookie.txt",
	}
	rotator, err := goproxyrotator.NewProxyRotator(proxies, cookieFiles, 60, 0, false)
````
*Note:* proxy URL has to be in form "protocol://username:password@proxy_address:port"
the function 
```go 
goproxyrotator.NewProxyRotator(proxy_list, cookie_files, timeout_secs, delayed_time, shuffle)
```` 
return a new rotator. The parameters:
- proxy_list is a string slice of proxy URLS (proxy URL has to be in form "protocol://username:password@proxy_address:port")
- cookie_files is a string slice of consistent cookie files (can be /path/file.txt). The number of cookie files has to be equal to the number of proxy URLs. A proxy URL in proxy slice will be paired with the corresponding cookie file in cookie file slice based on index of 2 slices.  "" means the corresponding proxy client does not use persistent cookie file. Cookie files can be shared,  many clients can use one same cookie file. 
- timed_out_secs sets timeout for clients (in Seconds).
- delayed_time (Milisecnonds). If delayed_time != 0, the next client got from rotator is based on time interval e.g. delayedTimeMsc = 1000, the next client will be fired every 1 second. If call GetProxyClient() many times within 1 sec, the same client will be returned.
- shuffle. If shuffle is true, every time the rotator finishes one round, it will shuffle the client list.

Method of ProxyClientRotator
```go 
(r *ProxyClientRotator) AddProxyClient(proxyURL, cookieFile string, timeoutSec int)
````
will add a new proxy client to the rotator.

To get the next client in the rotator call method:
```go 
(r *ProxyClientRotator) GetProxyClient()
````

To check health of all proxy HTTP clients, use:
```go 
(r *ProxyClientRotator) CheckHealthAll()
````
```CheckHealthAll()``` will return a map (key: proxyURL, value: true/fale)

**Important**: In order to save cookie to file. Trigger the method ```SaveCookie()``` of Enhanced Proxy Client (which is returned from ```GetProxyClient()```)