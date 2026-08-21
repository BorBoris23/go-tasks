package main

import (
	"strings"
	"sync"
)

type MyMicroservice struct {
	ACL               map[string][]string
	LogSubscribers    []Admin_LoggingServer
	StatisticsClients map[int]StatisticsClient
	ByMethod          map[string]uint64
	ByConsumer        map[string]uint64
	mu                sync.Mutex
}

type StatisticsClient struct {
	StatSubscribers Admin_StatisticsServer
	IntervalSeconds uint64

	ByMethod   map[string]uint64
	ByConsumer map[string]uint64
}

func (m *MyMicroservice) sendToLogSubscribers(event *Event) {
	m.mu.Lock()
	loggers := append([]Admin_LoggingServer(nil), m.LogSubscribers...)
	m.mu.Unlock()

	for _, logger := range loggers {
		logger.Send(event)
	}
}

func (m *MyMicroservice) countRequest(consumer, fullMethod string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ByMethod[fullMethod]++
	m.ByConsumer[consumer]++

	for _, client := range m.StatisticsClients {
		client.ByMethod[fullMethod]++
		client.ByConsumer[consumer]++
	}
}

func (m *MyMicroservice) isAllowed(consumer, fullMethod string) bool {
	for _, method := range m.ACL[consumer] {
		if method == fullMethod {
			return true
		}

		if strings.HasSuffix(method, "/*") {
			prefix := strings.TrimSuffix(method, "*")

			if strings.HasPrefix(fullMethod, prefix) {
				return true
			}
		}
	}

	return false
}
