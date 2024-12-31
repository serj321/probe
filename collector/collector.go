package collector

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type metric interface {
	Collect(*sql.DB, <-chan struct{}) error
	Name() string
	SetValue(string)
	GetValue() string
}

var _ metric = &openConnections{}
var _ metric = &slowQueries{}

type Collector struct {
	metrics  []metric
	dbCon    *sql.DB
	stopChan chan struct{}
}

func NewCollector(db *sql.DB) *Collector {
	return &Collector{
		metrics: []metric{
			&openConnections{},
			&slowQueries{},
		},
		dbCon:    db,
		stopChan: make(chan struct{}),
	}
}

func (c *Collector) StartCollecting() {
	var wg sync.WaitGroup
	for _, m := range c.metrics {
		wg.Add(1)
		go func(m metric) {
			defer wg.Done()
			m.Collect(c.dbCon, c.stopChan)
		}(m)
	}

	wg.Wait()
}

type openConnections struct {
	mu          sync.Mutex
	connections string
}

func (o *openConnections) Name() string {
	return "Open Connections"
}

func (o *openConnections) Collect(db *sql.DB, stopChan <-chan struct{}) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var openCons int
	for {
		select {
		case <-stopChan:
			fmt.Println("stopping connections collection...")
			return nil
		case <-ticker.C:
			err := db.QueryRow("SHOW STATUS LIKE 'threads_connected'").Scan(new(string), &openCons)
			if err != nil {
				return fmt.Errorf("failed to get threads connected count: %v", err)
			}
			fmt.Printf("threads currently connected: %v\n", openCons)
			o.SetValue(fmt.Sprintf("%v", openCons))
		}
	}
}

func (o *openConnections) GetValue() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.connections
}

func (o *openConnections) SetValue(connections string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connections = connections
}

type slowQueries struct {
	mu          sync.Mutex
	slowQueries string
}

func (s *slowQueries) Name() string {
	return "Slow Queries"
}

func (s *slowQueries) Collect(db *sql.DB, stopChan <-chan struct{}) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var slowQueries int
	for {
		select {
		case <-stopChan:
			fmt.Println("stopping slow queries collection...")
			return nil
		case <-ticker.C:
			err := db.QueryRow("show global status like 'slow_queries'").Scan(new(string), &slowQueries)
			if err != nil {
				return fmt.Errorf("failed to get slow queries: %v", err)
			}
			fmt.Printf("slow queries: %v\n", slowQueries)
		}
	}
}

func (s *slowQueries) GetValue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.slowQueries
}

func (s *slowQueries) SetValue(slowQueries string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slowQueries = slowQueries
}
