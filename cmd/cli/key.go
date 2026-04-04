package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage API keys",
}

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE:  runKeyList,
}

var keyCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeyCreate,
}

var keyDeleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "Delete an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeyDelete,
}

var keySetRateLimitCmd = &cobra.Command{
	Use:   "rate-limit [key]",
	Short: "Set rate limit for an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeySetRateLimit,
}

var (
	keyTTL          int
	keyRateRequests int
	keyRateWindow   int
)

func init() {
	keyCmd.AddCommand(keyListCmd, keyCreateCmd, keyDeleteCmd, keySetRateLimitCmd)

	keyCreateCmd.Flags().IntVar(&keyTTL, "ttl", 0, "TTL in days (0 = no expiration)")
	keySetRateLimitCmd.Flags().IntVar(&keyRateRequests, "requests", 100, "Max requests")
	keySetRateLimitCmd.Flags().IntVar(&keyRateWindow, "window", 60, "Window in seconds")
}

func runKeyList(cmd *cobra.Command, args []string) error {
	url := baseURL + "/api/keys"
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
		Status string `json:"status"`
		Keys   []struct {
			Name      string `json:"name"`
			KeyHash   string `json:"key_hash"`
			TTL       int    `json:"ttl_seconds"`
			RateLimit struct {
				Requests int `json:"requests"`
				Window   int `json:"window_seconds"`
			} `json:"rate_limit,omitempty"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Println("API Keys:")
	fmt.Println("=========")
	if len(result.Keys) == 0 {
		fmt.Println("  No keys found")
		return nil
	}

	for _, k := range result.Keys {
		ttl := "no expiration"
		if k.TTL > 0 {
			ttl = fmt.Sprintf("%d days", k.TTL/86400)
		}
		fmt.Printf("  %s\n", k.Name)
		fmt.Printf("    Hash: %s...\n", k.KeyHash[:min(16, len(k.KeyHash))])
		fmt.Printf("    TTL: %s\n", ttl)
		if k.RateLimit.Requests > 0 {
			fmt.Printf("    Rate Limit: %d req / %d sec\n", k.RateLimit.Requests, k.RateLimit.Window)
		}
		fmt.Println()
	}

	return nil
}

func runKeyCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	url := baseURL + "/api/keys"
	body := fmt.Sprintf(`{"name":"%s","ttl_days":%d}`, name, keyTTL)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create key: %s", string(respBody))
	}

	var result struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	fmt.Printf("API Key created: %s\n", result.Key)
	fmt.Println("Make sure to copy it - you won't be able to see it again!")
	return nil
}

func runKeyDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	url := baseURL + "/api/keys"
	body := fmt.Sprintf(`{"key":"%s"}`, key)

	req, err := http.NewRequest("DELETE", url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete key (status %d)", resp.StatusCode)
	}

	fmt.Printf("API Key deleted\n")
	return nil
}

func runKeySetRateLimit(cmd *cobra.Command, args []string) error {
	key := args[0]

	url := baseURL + "/api/keys/ratelimit"
	body := fmt.Sprintf(`{"key":"%s","requests":%d,"window_seconds":%d}`, key, keyRateRequests, keyRateWindow)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set rate limit (status %d)", resp.StatusCode)
	}

	fmt.Printf("Rate limit set: %d req / %d sec\n", keyRateRequests, keyRateWindow)
	return nil
}
