package main

import (
	"fmt"

	"github.com/Fordjour12/gcp-wif/internal/gcp"
)

func main() {
	fmt.Println("🧪 Testing GitHub OIDC Implementation")
	fmt.Println("====================================")

	// Test GitHub repository validation
	fmt.Println("\n📋 GitHub Repository Validation:")
	testRepos := map[string]bool{
		"owner/repo":          true,  // valid
		"owner-123/repo_test": true,  // valid with hyphens and underscores
		"invalid":             false, // no slash
		"owner/repo/extra":    false, // too many slashes
		".git/test":           false, // reserved name
	}

	for repo, expected := range testRepos {
		err := gcp.ValidateGitHubRepository(repo)
		isValid := err == nil
		status := "❌"
		if isValid == expected {
			status = "✅"
		}
		fmt.Printf("   %s %s (expected: %t, got: %t)\n", status, repo, expected, isValid)
	}

	// Test GitHub OIDC configuration
	fmt.Println("\n🔐 GitHub OIDC Configuration:")
	config := gcp.GetDefaultGitHubOIDCConfig()
	fmt.Printf("   ✅ Default Issuer URI: %s\n", config.IssuerURI)
	fmt.Printf("   ✅ Default Audiences: %v\n", config.AllowedAudiences)
	fmt.Printf("   ✅ Block Forked Repos: %t\n", config.BlockForkedRepos)
	fmt.Printf("   ✅ Require Actor: %t\n", config.RequireActor)

	// Test configuration validation
	err := gcp.ValidateGitHubOIDCConfig(config)
	if err != nil {
		fmt.Printf("   ❌ Configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Configuration validation passed\n")
	}

	// Test claims mapping
	fmt.Println("\n🏷️  GitHub Claims Mapping:")
	claims := gcp.GetDefaultGitHubClaimsMapping()
	fmt.Printf("   ✅ Subject: %s\n", claims.Subject)
	fmt.Printf("   ✅ Actor: %s\n", claims.Actor)
	fmt.Printf("   ✅ Repository: %s\n", claims.Repository)

	fmt.Println("\n🎉 GitHub OIDC Implementation Test Complete!")
}
