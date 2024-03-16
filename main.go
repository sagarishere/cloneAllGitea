package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	userReposEndpoint = "/api/v1/user/repos"
	timeout           = 5 * time.Minute
)

// Repository represents the basic structure of a Gitea repository
type Repository struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
	FullName string `json:"full_name"`
}

// Result is a struct to hold the result of a clone operation
type Result struct {
	RepoName string
	Err      error
}

func main() {
	config, err := loadConfig("config.env")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	giteaHost := config["GITEA_HOST"]
	giteaAccessToken := config["GITEA_ACCESS_TOKEN"]
	targetDir := config["TARGET_DIR"]

	// Ensure the target directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("Creating target directory: %s\n", targetDir)
		os.MkdirAll(targetDir, os.ModePerm)
	}

	// Change working directory to targetDir
	os.Chdir(targetDir)

	// Fetch repositories
	repos, err := fetchRepositories(giteaHost, giteaAccessToken)
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		return
	}

	// Total number of repositories
	fmt.Printf("Found %d repositories\n", len(repos))

	// Create a buffered channel with a capacity of the total number of repositories
	resultsCh := make(chan Result, len(repos))
	var wg sync.WaitGroup

	// Loop through repositories and clone if not already present, using goroutines
	for _, repo := range repos {
		wg.Add(1)
		go func(repo Repository) {
			defer wg.Done()
			// Create a new context with a timeout for each goroutine/request
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			if _, err := os.Stat(repo.FullName); !os.IsNotExist(err) {
				fmt.Printf("Repo %s already exists, skipping.\n", repo.FullName)
				resultsCh <- Result{RepoName: repo.FullName, Err: nil}
				return
			}

			fmt.Printf("Cloning %s from %s\n", repo.Name, repo.CloneURL)
			err := gitClone(ctx, repo.CloneURL, repo.FullName)
			resultsCh <- Result{RepoName: repo.FullName, Err: err}
		}(repo)
	}

	// Start a goroutine to close resultsCh when all goroutines are done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Process results as they come in
	for res := range resultsCh {
		if res.Err != nil {
			fmt.Printf("Error cloning repository %s: %v\n", res.RepoName, res.Err)
		}
	}
}

// fetchRepositories makes an API request to Gitea to fetch all repositories for the user
// and returns a list of repositories
func fetchRepositories(giteaHost, giteaAccessToken string) ([]Repository, error) {
	var allRepos []Repository
	client := &http.Client{}
	page := 1
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?page=%d", giteaHost, userReposEndpoint, page), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("Authorization", "token "+giteaAccessToken)
		response, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		if response.StatusCode != 200 {
			return nil, fmt.Errorf("API request failed with HTTP status code: %d", response.StatusCode)
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		var repos []Repository
		json.Unmarshal(body, &repos)

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}
	return allRepos, nil
}

// gitClone uses the git command to clone a repository given its URL
// It respects the provided context, allowing it to be canceled or timeout
func gitClone(ctx context.Context, cloneURL, addrToSave string) error {
	// Create a command with context
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, addrToSave)

	// Run the command and return any errors
	return cmd.Run()
}

func loadConfig(path string) (map[string]string, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make(map[string]string)
	lines := strings.Split(string(configFile), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue // skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad line in config file: %s", line)
		}
		config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return config, nil
}
