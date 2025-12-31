package hub

import (
	"fmt"
	"os"
	"path/filepath"
)

// HubClient interacts with the Monarch Registry (or mocks it)
type HubClient struct {
	BaseURL string
}

func NewHubClient() *HubClient {
	return &HubClient{
		BaseURL: "https://hub.monarch.dev/api/v1", // Mock URL
	}
}

// Search returns a list of available rulesets matching the query
func (c *HubClient) Search(query string) ([]string, error) {
	// Mock implementation
	// In reality, this would query an API
	mockResults := []string{
		"docker",
		"kubernetes-tools",
		"java-dev",
		"python-dev",
		"node-dev",
		"system-hardening",
		"gaming-setup",
	}

	var results []string
	for _, r := range mockResults {
		if query == "" || contains(r, query) {
			results = append(results, r)
		}
	}
	return results, nil
}

// Install downloads a ruleset and installs it into the destination directory
func (c *HubClient) Install(rulesetName, destDir string) error {
	// Mock implementation: Create a directory and a sample rules.yaml
	targetDir := filepath.Join(destDir, rulesetName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf("# RuleSet: %s\n# Downloaded from Monarch Hub\nresources:\n", rulesetName)

	// Create a dummy resource based on name for demo purposes
	if rulesetName == "docker" {
		content += `  - type: pkg
    params:
      name: docker
      state: present
  - type: service
    params:
      name: docker
      state: active
      enabled: true
`
	} else {
		content += "  []\n"
	}

	return os.WriteFile(filepath.Join(targetDir, "rules.yaml"), []byte(content), 0644)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr // Very accessible "contains" (prefix actually) for mock
}
