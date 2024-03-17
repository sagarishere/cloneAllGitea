package main

import (
	"context"
	"encoding/json"
	"flag"
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
	userEndpoint      = "/api/v1/user"
)

type Repository struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
	FullName string `json:"full_name"`
}

type Result struct {
	RepoName string
	Err      error
}

func main() {
	var (
		onlyMe bool
		user   string
	)
	flag.BoolVar(&onlyMe, "onlyme", false, "Fetch repositories owned by the user only")
	flag.StringVar(&user, "user", "", "Specify a username to fetch their repositories")
	flag.Parse()

	config, err := loadConfig("config.env")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	giteaHost := config["GITEA_HOST"]
	giteaAccessToken := config["GITEA_ACCESS_TOKEN"]
	targetDir := config["TARGET_DIR"]

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("Creating target directory: %s\n", targetDir)
		os.MkdirAll(targetDir, os.ModePerm)
	}

	os.Chdir(targetDir)

	var username string
	if onlyMe {
		username, err = fetchUsername(giteaHost, giteaAccessToken)
		if err != nil {
			fmt.Printf("Error fetching user details: %v\n", err)
			return
		}
	} else if user != "" {
		username = user
	}

	repos, err := fetchRepositories(giteaHost, giteaAccessToken, username, onlyMe || user != "")
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		return
	}

	fmt.Printf("Found %d repositories\n", len(repos))

	resultsCh := make(chan Result, len(repos))
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(repo Repository) {
			defer wg.Done()
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

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for res := range resultsCh {
		if res.Err != nil {
			fmt.Printf("Error cloning repository %s: %v\n", res.RepoName, res.Err)
		}
	}
}

func fetchRepositories(giteaHost, giteaAccessToken, username string, filterByUsername bool) ([]Repository, error) {
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

		if filterByUsername && username != "" {
			for _, repo := range repos {
				if strings.Split(repo.FullName, "/")[0] == username {
					allRepos = append(allRepos, repo)
				}
			}
		} else {
			allRepos = append(allRepos, repos...)
		}

		page++
	}
	return allRepos, nil
}

func gitClone(ctx context.Context, cloneURL, addrToSave string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, addrToSave)
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
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad line in config file: %s", line)
		}
		config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return config, nil
}

func fetchUsername(giteaHost, giteaAccessToken string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", giteaHost, userEndpoint), nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "token "+giteaAccessToken)
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch user details with status code: %d", response.StatusCode)
	}

	var userDetails struct {
		Username string `json:"login"`
	}
	err = json.NewDecoder(response.Body).Decode(&userDetails)
	if err != nil {
		return "", err
	}

	return userDetails.Username, nil
}
