package main

import (
	"context"
	"fmt"
	"mobetta/internal/util"
	"os"
	"sync"
)

func main() {
	// Get the list of repositories from the command line arguments.
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <destination_directory> <file_containing_repos>")
		fmt.Println("Example: go run main.go /tmp/repos /tmp/github_repos.txt")
		os.Exit(1)
	}

	destinationDir := os.Args[1]
	repoURLFile := os.Args[2]

	repoURLs, err := util.GetRepoURLs(repoURLFile)
	if err != nil {
		fmt.Printf("Error reading repo URLs from file: %v\n", err)
		os.Exit(1)
	}

	// Create the destination directory if it doesn't exist.
	if _, err := os.Stat(destinationDir); os.IsNotExist(err) {
		if err := os.MkdirAll(destinationDir, 0755); err != nil {
			fmt.Printf("Error creating destination directory: %v\n", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, len(repoURLs)) // Collect errors from goroutines.
	barrier := make(chan struct{}, 16)

	// Clone each repository in parallel.
	for _, repoURL := range repoURLs {
		barrier <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer func() {
				wg.Done()
				<-barrier
			}()
			repoName := util.GetRepoName(url)
			if repoName == "" {
				errCh <- fmt.Errorf("invalid repo URL: %s", url)
				return
			}
			repoOrg := util.GetRepoOrg(url)
			if repoOrg == "" {
				errCh <- fmt.Errorf("invalid repo URL: %s", url)
				return
			}
			dest := fmt.Sprintf("%s/%s/%s", destinationDir, repoOrg, repoName)
			err := util.CloneRepo(ctx, url, dest)
			if err != nil {
				errCh <- err // Send error to the channel
			}
		}(repoURL)
	}

	wg.Wait()    // Wait for all clones to complete.
	close(errCh) // Close the error channel.

	// Check for errors.
	hasErrors := false
	for err := range errCh {
		fmt.Printf("Error: %v\n", err) // Print any errors that occurred
		hasErrors = true
	}

	if hasErrors {
		fmt.Println("Some repositories failed to clone.")
		os.Exit(1)
	}

	fmt.Println("All repositories cloned successfully.")
}
