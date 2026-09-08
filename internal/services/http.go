package services

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func probeHTTP(services []Service) {
	client := &http.Client{
		Timeout:   300 * time.Millisecond,
		Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	results := make(map[int]bool)
	var mutex sync.Mutex
	var wait sync.WaitGroup

	for i := range services {
		for _, port := range services[i].Ports {
			if port.Host == 0 || port.Protocol != "tcp" {
				continue
			}
			mutex.Lock()
			if _, exists := results[port.Host]; exists {
				mutex.Unlock()
				continue
			}
			results[port.Host] = false
			mutex.Unlock()
			wait.Add(1)
			go func(port int) {
				defer wait.Done()
				if !isHTTP(client, port) {
					return
				}
				mutex.Lock()
				results[port] = true
				mutex.Unlock()
			}(port.Host)
		}
	}
	wait.Wait()

	for serviceIndex := range services {
		for portIndex := range services[serviceIndex].Ports {
			if results[services[serviceIndex].Ports[portIndex].Host] {
				services[serviceIndex].Ports[portIndex].Scheme = "http"
			}
		}
	}
}

func isHTTP(client *http.Client, port int) bool {
	for _, host := range []string{"127.0.0.1", "[::1]"} {
		request, err := http.NewRequest(http.MethodHead, fmt.Sprintf("http://%s:%d/", host, port), nil)
		if err != nil {
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			continue
		}
		_ = response.Body.Close()
		return true
	}
	return false
}
