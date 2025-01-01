package collector

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Metric interface {
	Collect(*sql.DB, <-chan struct{}) error
	Name() string
	SetValue(string)
	GetValue() string
}

var _ Metric = &OpenConnections{}
var _ Metric = &SlowQueries{}

type Collector struct {
	metrics  []Metric
	dbCon    *sql.DB
	stopChan chan struct{}
}

func NewCollector(db *sql.DB, metrics []Metric, stopChan chan struct{}) *Collector {
	return &Collector{
		metrics:  metrics,
		dbCon:    db,
		stopChan: stopChan,
	}
}

func (c *Collector) StartCollecting() {
	var wg sync.WaitGroup
	for _, m := range c.metrics {
		wg.Add(1)
		go func(m Metric) {
			defer wg.Done()
			m.Collect(c.dbCon, c.stopChan)
		}(m)
	}

	wg.Wait()
}

type OpenConnections struct {
	mu          sync.Mutex
	connections string
}

func (o *OpenConnections) Name() string {
	return "Open Connections"
}

func (o *OpenConnections) Collect(db *sql.DB, stopChan <-chan struct{}) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var openCons int
	for {
		select {
		case <-stopChan:
			fmt.Println("Stopping connections collection...")
			return nil
		case <-ticker.C:
			err := db.QueryRow("SHOW STATUS LIKE 'threads_connected'").Scan(new(string), &openCons)
			if err != nil {
				return fmt.Errorf("failed to get threads connected count: %v", err)
			}
			o.SetValue(fmt.Sprintf("%v", openCons))
		}
	}
}

func (o *OpenConnections) GetValue() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.connections
}

func (o *OpenConnections) SetValue(connections string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connections = connections
}

type SlowQueries struct {
	mu          sync.Mutex
	slowQueries string
}

func (s *SlowQueries) Name() string {
	return "Slow Queries"
}

func (s *SlowQueries) Collect(db *sql.DB, stopChan <-chan struct{}) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var slowQueries int
	for {
		select {
		case <-stopChan:
			fmt.Println("Stopping slow queries collection...")
			return nil
		case <-ticker.C:
			err := db.QueryRow("show global status like 'slow_queries'").Scan(new(string), &slowQueries)
			if err != nil {
				return fmt.Errorf("failed to get slow queries: %v", err)
			}
			s.SetValue(fmt.Sprintf("%v", slowQueries))
		}
	}
}

func (s *SlowQueries) GetValue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.slowQueries
}

func (s *SlowQueries) SetValue(slowQueries string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slowQueries = slowQueries
}
