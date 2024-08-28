package util

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GitHubFile struct {
	Content string `json:"content"`
}

type GitHubPath struct {
	Path string `json:"path"`
}

func FetchGithubFile(owner, repo, branch, path, token string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Failed to fetch file: %s", resp.Status))
	}

	var result []string

	body, _ := ioutil.ReadAll(resp.Body)

	var file GitHubFile
	err = json.Unmarshal(body, &file)
	if err != nil {
		var githubPaths []GitHubPath
		err = json.Unmarshal(body, &githubPaths)
		if err != nil {
			return nil, err
		}
		for _, githubPath := range githubPaths {
			content, err := FetchGithubFile(owner, repo, branch, githubPath.Path, token)
			if err != nil {
				return nil, err
			}

			result = append(result, content...)
		}
	} else {
		decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, err
		}
		result = append(result, string(decodedContent))
	}

	return result, err
}
