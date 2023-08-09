package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type TokenResponse struct {
	Token string `json:"token"`
}

func fetchBearerToken(authenticateHeader, username, password string) (string, error) {
	// Extract necessary info from the WWW-Authenticate header
	realm := strings.Split(authenticateHeader, "realm=\"")[1]
	realm = strings.Split(realm, "\"")[0]

	service := strings.Split(authenticateHeader, "service=\"")[1]
	service = strings.Split(service, "\"")[0]

	scope := strings.Split(authenticateHeader, "scope=\"")[1]
	scope = strings.Split(scope, "\"")[0]

	tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)

	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", err
	}

	// If credentials are provided, set them
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch bearer token, status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", err
	}

	return tokenResponse.Token, nil
}

func main() {
	// Command line flags for authentication
	var username, password string
	flag.StringVar(&username, "user", "", "Username for Docker registry authentication")
	flag.StringVar(&password, "pass", "", "Password for Docker registry authentication")

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("You must provide the image and tag as 'image:tag' or 'registry/namespace/image:tag'.")
		os.Exit(1)
	}

	// Split the image:tag format
	imageParts := strings.Split(args[0], ":")
	if len(imageParts) != 2 {
		fmt.Println("Invalid format. Expected 'image:tag' or 'registry/namespace/image:tag'.")
		os.Exit(1)
	}

	registryURL := "https://registry-1.docker.io" // default to Docker Hub
	repoName, tag := imageParts[0], imageParts[1]

	// If a different registry is specified, extract it from the image reference
	if strings.Count(repoName, "/") > 1 {
		parts := strings.SplitN(repoName, "/", 2)
		registryURL = "https://" + parts[0]
		repoName = parts[1]
	}

	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, repoName, tag)

	// Set headers required by the Docker registry API
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		os.Exit(1)
	}

	// Basic Auth, if credentials are provided
	if username != "" && password != "" {
		auth := username + ":" + password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Add("Authorization", "Basic "+encodedAuth)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// If a 401 Unauthorized response is received, try to fetch a bearer token
	if resp.StatusCode == 401 {
		token, err := fetchBearerToken(resp.Header.Get("WWW-Authenticate"), username, password)
		if err != nil {
			fmt.Println("Failed to get bearer token:", err)
			os.Exit(1)
		}

		// Retry the request with the bearer token
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

		resp, err = client.Do(req)
		if err != nil {
			fmt.Println("Error making request:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
	}

	// 200-299 statuses indicate that the tag exists
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		fmt.Println("exist")
		os.Exit(0)
	}

	// If tag is not found, exit 4
	if resp.StatusCode == 404 {
		fmt.Println("noexist")
		os.Exit(0)
	}

	// Any other status is an error
	fmt.Printf("Unexpected response: %s\n", resp.Status)
	os.Exit(1)
}
