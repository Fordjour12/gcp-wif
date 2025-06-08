package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourusername/wif-setup/internal/wif"
)

var rootCmd = &cobra.Command{
	Use:   "wif-setup",
	Short: "Simple tool to setup GitHub Actions → GCP Workload Identity Federation",
	Long:  `A focused CLI tool that sets up Workload Identity Federation between GitHub Actions and Google Cloud Platform in one command.`,
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup WIF for GitHub Actions",
	Long:  `Creates workload identity pool, provider, and service account bindings for GitHub Actions authentication.`,
	RunE:  runSetup,
}

var (
	project    string
	repository string
	saEmail    string
	poolID     string
	providerID string
)

func init() {
	setupCmd.Flags().StringVarP(&project, "project", "p", "", "GCP Project ID (required)")
	setupCmd.Flags().StringVarP(&repository, "repo", "r", "", "GitHub repository (owner/repo format, required)")
	setupCmd.Flags().StringVarP(&saEmail, "service-account", "s", "", "Service account email (optional, will be created if not provided)")
	setupCmd.Flags().StringVar(&poolID, "pool-id", "", "Workload Identity Pool ID (optional, auto-generated)")
	setupCmd.Flags().StringVar(&providerID, "provider-id", "github-provider", "Workload Identity Provider ID")

	setupCmd.MarkFlagRequired("project")
	setupCmd.MarkFlagRequired("repo")

	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	config := &wif.Config{
		Project:    project,
		Repository: repository,
		SAEmail:    saEmail,
		PoolID:     poolID,
		ProviderID: providerID,
	}

	// Validate and set defaults
	if err := config.SetDefaults(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Print configuration
	fmt.Println("🚀 Setting up Workload Identity Federation")
	fmt.Printf("   Project: %s\n", config.Project)
	fmt.Printf("   Repository: %s\n", config.Repository)
	fmt.Printf("   Pool ID: %s\n", config.PoolID)
	fmt.Printf("   Provider ID: %s\n", config.ProviderID)
	fmt.Printf("   Service Account: %s\n", config.SAEmail)
	fmt.Println()

	// Run setup
	client := wif.NewClient(config.Project)
	return client.Setup(config)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
