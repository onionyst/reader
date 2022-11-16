package utils

import (
	"net"
	"sync"
	"time"
)

// Wait waits services to hold for seconds
func Wait(services []string, sec int) bool {
	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(services))

	success := make(chan bool, 1)

	go func() {
		for _, service := range services {
			go waitOne(service, &wg, now)
		}
		wg.Wait()
		success <- true
	}()

	select {
	case <-success:
		return true
	case <-time.After(time.Duration(sec) * time.Second):
		return false
	}
}

func waitOne(service string, wg *sync.WaitGroup, start time.Time) {
	defer wg.Done()
	for {
		if _, err := net.Dial("tcp", service); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
}
