package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	apiURL = "https://sourcegraph.com/.api" // Replace with your Sourcegraph instance URL
)

type Repository struct {
	Name string `json:"name"`
}

type Organization struct {
	Name string `json:"name"`
}

type GraphQLResponse struct {
	Data struct {
		Organizations struct {
			Nodes []Organization `json:"nodes"`
		} `json:"organizations"`
		Organization struct {
			Repositories struct {
				Nodes []Repository `json:"nodes"`
			} `json:"repositories"`
		} `json:"organization"`
		Repository struct {
			File struct {
				IsDirectory bool `json:"isDirectory"`
			} `json:"file"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func main() {
	orgs, err := getOrganizations()
	if err != nil {
		fmt.Println("Error fetching organizations:", err)
		os.Exit(1)
	}

	for _, org := range orgs {
		fmt.Println("Organization:", org.Name)
		repos, err := getRepositories(org.Name)
		if err != nil {
			fmt.Println("Error fetching repositories for organization", org.Name, ":", err)
			continue
		}
		for _, repo := range repos {
			hasGitpodFile, err := checkGitpodFile(repo.Name)
			if err != nil {
				fmt.Println("Error checking .gitpod.yml for repository", repo.Name, ":", err)
				continue
			}
			if hasGitpodFile {
				fmt.Println("  - Repository:", repo.Name)
			}
		}
	}
}

func getOrganizations() ([]Organization, error) {
	var response GraphQLResponse
	query := `{
		organizations(first: 100) {
			nodes {
				name
			}
		}
	}`
	err := makeGraphQLRequest(query, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.Organizations.Nodes, nil
}

func getRepositories(orgName string) ([]Repository, error) {
	var response GraphQLResponse
	query := fmt.Sprintf(`{
		organization(name: "%s") {
			repositories(first: 100) {
				nodes {
					name
				}
			}
		}
	}`, orgName)
	err := makeGraphQLRequest(query, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.Organization.Repositories.Nodes, nil
}

func checkGitpodFile(repoName string) (bool, error) {
	var response GraphQLResponse
	query := fmt.Sprintf(`{
		repository(name: "%s") {
			file(name: ".gitpod.yml") {
				isDirectory
			}
		}
	}`, repoName)
	err := makeGraphQLRequest(query, &response)
	if err != nil {
		return false, err
	}
	// Assumes that if the file field is present, then the file exists
	return !response.Data.Repository.File.IsDirectory, nil
}

func makeGraphQLRequest(query string, response *GraphQLResponse) error {

	accessToken := os.Getenv("SOURCEGRAPH_TOKEN") // Replace with your Sourcegraph access token

	client := &http.Client{}
	reqBody, err := json.Marshal(map[string]string{
		"query": query,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", apiURL+"/graphql", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, body)
	}
	err = json.Unmarshal(body, response)
	if err != nil {
		return err
	}
	if len(response.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", response.Errors)
	}
	return nil
}
