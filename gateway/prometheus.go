package gateway

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"proxymesh/internal/config"
)

type PrometheusPusher struct {
	enabled    bool
	gatewayURL string
	interval   time.Duration
	jobName    string
	stopCh     chan struct{}
}

func NewPrometheusPusher(cfg *config.PrometheusConfig) *PrometheusPusher {
	if !cfg.Enabled {
		return &PrometheusPusher{enabled: false}
	}

	return &PrometheusPusher{
		enabled:    true,
		gatewayURL: cfg.PushGatewayURL,
		interval:   time.Duration(cfg.PushIntervalSeconds) * time.Second,
		jobName:    cfg.JobName,
		stopCh:     make(chan struct{}),
	}
}

func (p *PrometheusPusher) Start() {
	if !p.enabled {
		return
	}

	go p.run()
}

func (p *PrometheusPusher) run() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.push()
		case <-p.stopCh:
			p.push()
			return
		}
	}
}

func (p *PrometheusPusher) push() {
	if p.gatewayURL == "" {
		return
	}

	metricsOutput := metrics.String()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/metrics/job/%s", p.gatewayURL, p.jobName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(metricsOutput))
	if err != nil {
		Error("Failed to create prometheus push request", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		Error("Failed to push to prometheus gateway", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		Warn("Prometheus gateway returned error", map[string]interface{}{
			"status": resp.StatusCode,
		})
	}
}

func (p *PrometheusPusher) Stop() {
	if !p.enabled {
		return
	}
	close(p.stopCh)
}
