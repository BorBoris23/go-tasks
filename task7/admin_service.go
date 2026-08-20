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
	statisticsClients := StatisticsClient{
		StatSubscribers: statistics,
		IntervalSeconds: start.IntervalSeconds,
		ByMethod:        make(map[string]uint64),
		ByConsumer:      make(map[string]uint64),
	}

	m.microservice.StatisticsClients = append(
		m.microservice.StatisticsClients,
		statisticsClients,
	)

	clientIndex := len(m.microservice.StatisticsClients) - 1

	m.microservice.mu.Unlock()

	go m.microservice.sendStatistics(clientIndex)

	<-statistics.Context().Done()

	return nil
}

func (m *MyMicroservice) sendStatistics(index int) {
	m.mu.Lock()
	statistics := m.StatisticsClients[index].StatSubscribers
	interval := m.StatisticsClients[index].IntervalSeconds
	m.mu.Unlock()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()

			byMethod := make(map[string]uint64)
			for method, count := range m.StatisticsClients[index].ByMethod {
				byMethod[method] = count
			}

			byConsumer := make(map[string]uint64)
			for consumer, count := range m.StatisticsClients[index].ByConsumer {
				byConsumer[consumer] = count
			}

			m.StatisticsClients[index].ByMethod = make(map[string]uint64)
			m.StatisticsClients[index].ByConsumer = make(map[string]uint64)

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
