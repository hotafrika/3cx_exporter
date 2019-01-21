package exporter

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const prefix = "pbx_"

var (
	blacklistSizeDesc        = prometheus.NewDesc(prefix+"blacklist_size", "Number of blacklisted IP addresses", nil, nil)
	callsActiveDesc          = prometheus.NewDesc(prefix+"calls_active", "Number of current active calls", nil, nil)
	callsLimitDesc           = prometheus.NewDesc(prefix+"calls_limit", "Maximum number of supported simultaneous calls", nil, nil)
	extensionsTotalDesc      = prometheus.NewDesc(prefix+"extensions_total", "Number of total extensions", nil, nil)
	extensionsRegisteredDesc = prometheus.NewDesc(prefix+"extensions_registered", "Number of registered extensions", nil, nil)
	backupAgeDesc            = prometheus.NewDesc(prefix+"backup_age", "Age of last backup in seconds", nil, nil)
	maintenanceRemainingDesc = prometheus.NewDesc(prefix+"maintenance_remaining", "Remaining time of maintenance in seconds", nil, nil)

	serviceStatusDesc = prometheus.NewDesc(prefix+"service_status", "Status of service", []string{"name"}, nil)
	serviceCPUDesc    = prometheus.NewDesc(prefix+"service_cpu", "CPU usage of service", []string{"name"}, nil)
	serviceMemoryDesc = prometheus.NewDesc(prefix+"service_memory", "Memory usage of service", []string{"name"}, nil)

	trunkRegisteredDesc = prometheus.NewDesc(prefix+"trunk_registered", "Status of trunk", []string{"name"}, nil)
)

// Exporter represents a prometheus exporter
type Exporter struct {
	API
}

// Describe describes the metrics
func (ex *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- blacklistSizeDesc
	ch <- callsActiveDesc
	ch <- callsLimitDesc
	ch <- extensionsTotalDesc
	ch <- extensionsRegisteredDesc
	ch <- backupAgeDesc
	ch <- maintenanceRemainingDesc

	ch <- serviceStatusDesc
	ch <- serviceCPUDesc
	ch <- serviceMemoryDesc

	ch <- trunkRegisteredDesc
}

// Collect collects the metrics
func (ex *Exporter) Collect(ch chan<- prometheus.Metric) {
	now := time.Now()

	status, err := ex.API.SystemStatus()
	if err == ErrAuthentication {
		log.Println("authentication failed:", err)
		return
	}
	if err == nil {
		ch <- prometheus.MustNewConstMetric(blacklistSizeDesc, prometheus.GaugeValue, float64(status.BlacklistedIPCount))
		ch <- prometheus.MustNewConstMetric(callsActiveDesc, prometheus.GaugeValue, float64(status.CallsActive))
		ch <- prometheus.MustNewConstMetric(callsLimitDesc, prometheus.GaugeValue, float64(status.MaxSimCalls))
		ch <- prometheus.MustNewConstMetric(extensionsTotalDesc, prometheus.GaugeValue, float64(status.ExtensionsTotal))
		ch <- prometheus.MustNewConstMetric(extensionsRegisteredDesc, prometheus.GaugeValue, float64(status.ExtensionsRegistered))

		// seconds since last backup
		backupAgo := float64(-1)
		if t := status.LastBackupDateTime; t != nil {
			backupAgo = float64(now.Sub(*t)) / float64(time.Second)
		}
		ch <- prometheus.MustNewConstMetric(backupAgeDesc, prometheus.CounterValue, backupAgo)

		// remaining time of maintenance
		maintenanceRemaining := float64(-1)
		if t := status.MaintenanceExpiresAt; t != nil {
			maintenanceRemaining = float64(t.Sub(now)) / float64(time.Second)
		}
		ch <- prometheus.MustNewConstMetric(maintenanceRemainingDesc, prometheus.CounterValue, maintenanceRemaining)
	} else {
		log.Println("failed to fetch SystemStatus:", err)
	}

	services, err := ex.API.ServiceList()
	if err == nil {
		for i := range services {
			service := services[i]
			labels := []string{service.Name}

			ch <- prometheus.MustNewConstMetric(serviceStatusDesc, prometheus.GaugeValue, float64(service.Status), labels...)
			ch <- prometheus.MustNewConstMetric(serviceCPUDesc, prometheus.GaugeValue, float64(service.CPUUsage), labels...)
			ch <- prometheus.MustNewConstMetric(serviceMemoryDesc, prometheus.GaugeValue, float64(service.MemoryUsed), labels...)
		}
	} else {
		log.Println("failed to fetch ServiceList:", err)
	}

	trunks, err := ex.API.TrunkList()
	if err == nil {
		for i := range trunks {
			trunk := trunks[i]
			labels := []string{trunk.Name}

			registered := 0
			if trunk.IsRegistered {
				registered = 1
			}
			ch <- prometheus.MustNewConstMetric(trunkRegisteredDesc, prometheus.GaugeValue, float64(registered), labels...)
		}
	} else {
		log.Println("failed to fetch TrunkList:", err)
	}
}
