package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show gateway status and metrics",
	RunE:  runStatus,
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check gateway health",
	RunE:  runHealth,
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show gateway metrics",
	RunE:  runMetrics,
}

var capacityCmd = &cobra.Command{
	Use:   "capacity",
	Short: "Show capacity report",
	RunE:  runCapacity,
}

func init() {
	statusCmd.AddCommand(healthCmd, metricsCmd, capacityCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if err := runHealth(cmd, nil); err != nil {
		return err
	}
	fmt.Println()
	if err := runMetrics(cmd, nil); err != nil {
		return err
	}
	return nil
}

func runHealth(cmd *cobra.Command, args []string) error {
	url := baseURL + "/health"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway unhealthy (status %d)", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Println("Gateway Status:")
	fmt.Println("===============")
	fmt.Printf("  Status: %s\n", result["status"])
	fmt.Printf("  Version: %s\n", result["version"])
	return nil
}

func runMetrics(cmd *cobra.Command, args []string) error {
	url := baseURL + "/metrics"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Metrics:")
	fmt.Println("========")
	fmt.Println(string(body))
	return nil
}

func runCapacity(cmd *cobra.Command, args []string) error {
	url := baseURL + "/api/admin/capacity"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Admin-Key", adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status         string  `json:"status"`
		TotalNodes     int     `json:"total_nodes"`
		HealthyNodes   int     `json:"healthy_nodes"`
		UnhealthyNodes int     `json:"unhealthy_nodes"`
		CriticalNodes  int     `json:"critical_nodes"`
		WarningNodes   int     `json:"warning_nodes"`
		AvgUtilization float64 `json:"avg_utilization"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Println("Capacity Report:")
	fmt.Println("=================")
	fmt.Printf("  Total Nodes:     %d\n", result.TotalNodes)
	fmt.Printf("  Healthy:        %d\n", result.HealthyNodes)
	fmt.Printf("  Unhealthy:     %d\n", result.UnhealthyNodes)
	fmt.Printf("  Critical:       %d\n", result.CriticalNodes)
	fmt.Printf("  Warning:       %d\n", result.WarningNodes)
	fmt.Printf("  Avg Utilization: %.1f%%\n", result.AvgUtilization)
	return nil
}
