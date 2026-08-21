package main

import "time"

type AdminService struct {
	UnimplementedAdminServer
	microservice *MyMicroservice
}

func (m *AdminService) Logging(n *Nothing, logger Admin_LoggingServer) error {
	m.microservice.mu.Lock()
	m.microservice.LogSubscribers = append(m.microservice.LogSubscribers, logger)
	m.microservice.mu.Unlock()

	<-logger.Context().Done()

	return nil
}

func (m *AdminService) Statistics(start *StatInterval, statistics Admin_StatisticsServer) error {

	m.microservice.mu.Lock()

	statisticsClient := StatisticsClient{
		StatSubscribers: statistics,
		IntervalSeconds: start.IntervalSeconds,
		ByMethod:        make(map[string]uint64),
		ByConsumer:      make(map[string]uint64),
	}

	clientID := len(m.microservice.StatisticsClients)

	m.microservice.StatisticsClients[clientID] = statisticsClient

	m.microservice.mu.Unlock()

	go m.microservice.sendStatistics(clientID)

	<-statistics.Context().Done()

	m.microservice.mu.Lock()
	delete(m.microservice.StatisticsClients, clientID)
	m.microservice.mu.Unlock()

	return nil
}

func (m *MyMicroservice) sendStatistics(clientID int) {
	m.mu.Lock()

	client, ok := m.StatisticsClients[clientID]
	if !ok {
		m.mu.Unlock()
		return
	}

	statistics := client.StatSubscribers
	interval := client.IntervalSeconds

	m.mu.Unlock()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()

			client, ok := m.StatisticsClients[clientID]
			if !ok {
				m.mu.Unlock()
				return
			}

			byMethod := make(map[string]uint64)
			for method, count := range client.ByMethod {
				byMethod[method] = count
			}

			byConsumer := make(map[string]uint64)
			for consumer, count := range client.ByConsumer {
				byConsumer[consumer] = count
			}

			m.StatisticsClients[clientID] = StatisticsClient{
				StatSubscribers: client.StatSubscribers,
				IntervalSeconds: client.IntervalSeconds,
				ByMethod:        make(map[string]uint64),
				ByConsumer:      make(map[string]uint64),
			}

			m.mu.Unlock()

			stat := &Stat{
				Timestamp:  time.Now().Unix(),
				ByMethod:   byMethod,
				ByConsumer: byConsumer,
			}

			if err := statistics.Send(stat); err != nil {
				return
			}

		case <-statistics.Context().Done():
			return
		}
	}
}
