package util

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// cloneRepo clones a single repository.
func CloneRepo(ctx context.Context, repoURL, destinationDir string) error {
	fmt.Printf("Cloning %s into %s...\n", repoURL, destinationDir)

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, destinationDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error cloning %s: %s\nOutput: %s\n", repoURL, err, output)
		return fmt.Errorf("failed to clone %s: %w, output: %s", repoURL, err, output)
	}
	fmt.Printf("Successfully cloned %s\n", repoURL)
	return nil
}

// getRepoName extracts the repository name from the full URL.
func GetRepoName(repoURL string) string {
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func GetRepoOrg(repoURL string) string {
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-2]
	}
	return ""
}

func GetRepoURLs(repoURLFile string) ([]string, error) {
	file, err := os.Open(repoURLFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()
	repoURLs := make([]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repoURLs = append(repoURLs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err.Error())
	}
	return repoURLs, nil
}
