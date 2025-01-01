package displayer

import (
	"fmt"
	"probe/collector"
	"time"
)

func StartDisplaying(metrics []collector.Metric, stopChan <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// give the collector a chance to get some metrics
	time.Sleep(4 * time.Second)

	for {
		select {
		case <-stopChan:
			fmt.Println("Stopping display go routine...")
			return
		case <-ticker.C:
			for _, m := range metrics {
				mValue := m.GetValue()
				mName := m.Name()
				fmt.Printf("%v: %v\n", mName, mValue)
			}
		}
	}
}
