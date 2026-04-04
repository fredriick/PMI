package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	adminKey string
	baseURL  string
	cfgFile  string
)

var rootCmd = &cobra.Command{
	Use:   "proxymesh-cli",
	Short: "ProxyMesh CLI for node and key management",
	Long:  `ProxyMesh CLI - Manage nodes, API keys, and query network status`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&adminKey, "admin-key", "test-admin-key", "Admin API key")
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8000", "Gateway base URL")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (optional)")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ProxyMesh CLI v1.0.0")
	},
}

func main() {
	rootCmd.AddCommand(nodeCmd, keyCmd, statusCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
