package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage proxy nodes",
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered nodes",
	RunE:  runNodeList,
}

var nodeRegisterCmd = &cobra.Command{
	Use:   "register [node_id]",
	Short: "Register a new node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodeRegister,
}

var nodeDeregisterCmd = &cobra.Command{
	Use:   "deregister [node_id]",
	Short: "Deregister a node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodeDeregister,
}

var nodeStatusCmd = &cobra.Command{
	Use:   "status [node_id]",
	Short: "Get status of a specific node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodeStatus,
}

var (
	nodeType    string
	nodeIP      string
	nodeCountry string
	nodeCity    string
	nodeISP     string
	nodeOS      string
)

func init() {
	nodeCmd.AddCommand(nodeListCmd, nodeRegisterCmd, nodeDeregisterCmd, nodeStatusCmd)

	nodeRegisterCmd.Flags().StringVar(&nodeType, "type", "datacenter", "Node type (datacenter/residential)")
	nodeRegisterCmd.Flags().StringVar(&nodeIP, "ip", "", "Node IP address")
	nodeRegisterCmd.Flags().StringVar(&nodeCountry, "country", "US", "Country code")
	nodeRegisterCmd.Flags().StringVar(&nodeCity, "city", "", "City name")
	nodeRegisterCmd.Flags().StringVar(&nodeISP, "isp", "", "ISP name")
	nodeRegisterCmd.Flags().StringVar(&nodeOS, "os", "linux", "Operating system")
}

func runNodeList(cmd *cobra.Command, args []string) error {
	url := baseURL + "/api/admin/nodes"
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		Nodes  []struct {
			ID   string `json:"id"`
			Node struct {
				NodeID   string `json:"node_id"`
				Type     string `json:"node_type"`
				Country  string `json:"country"`
				City     string `json:"city"`
				ISP      string `json:"isp"`
				IP       string `json:"ip"`
				OS       string `json:"os"`
				LastSeen string `json:"last_seen"`
			} `json:"node"`
		} `json:"nodes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Println("Registered Nodes:")
	fmt.Println("=================")
	if len(result.Nodes) == 0 {
		fmt.Println("  No nodes registered")
		return nil
	}

	for _, n := range result.Nodes {
		fmt.Printf("  %s (%s)\n", n.ID, n.Node.Type)
		fmt.Printf("    Country: %s | City: %s | ISP: %s\n", n.Node.Country, n.Node.City, n.Node.ISP)
		fmt.Printf("    IP: %s | OS: %s | Last Seen: %s\n", n.Node.IP, n.Node.OS, n.Node.LastSeen)
		fmt.Println()
	}

	return nil
}

func runNodeRegister(cmd *cobra.Command, args []string) error {
	nodeID := args[0]

	url := baseURL + "/api/admin/nodes"
	body := fmt.Sprintf(`{"node_id":"%s","node_type":"%s","ip":"%s","country":"%s","city":"%s","isp":"%s","os":"%s"}`,
		nodeID, nodeType, nodeIP, nodeCountry, nodeCity, nodeISP, nodeOS)

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
		return fmt.Errorf("failed to register node: %s", string(respBody))
	}

	fmt.Printf("Node '%s' registered successfully\n", nodeID)
	return nil
}

func runNodeDeregister(cmd *cobra.Command, args []string) error {
	nodeID := args[0]

	url := baseURL + "/api/admin/nodes/" + nodeID
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Admin-Key", adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to deregister node (status %d)", resp.StatusCode)
	}

	fmt.Printf("Node '%s' deregistered successfully\n", nodeID)
	return nil
}

func runNodeStatus(cmd *cobra.Command, args []string) error {
	nodeID := args[0]

	url := baseURL + "/api/admin/nodes/" + nodeID
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("node not found (status %d)", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		ID     string `json:"id"`
		Node   struct {
			NodeID   string `json:"node_id"`
			Type     string `json:"node_type"`
			Country  string `json:"country"`
			City     string `json:"city"`
			ISP      string `json:"isp"`
			IP       string `json:"ip"`
			OS       string `json:"os"`
			LastSeen string `json:"last_seen"`
		} `json:"node"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("Node: %s\n", result.ID)
	fmt.Printf("  Type: %s | Country: %s | City: %s\n", result.Node.Type, result.Node.Country, result.Node.City)
	fmt.Printf("  ISP: %s | IP: %s\n", result.Node.ISP, result.Node.IP)
	fmt.Printf("  OS: %s | Last Seen: %s\n", result.Node.OS, result.Node.LastSeen)

	return nil
}
