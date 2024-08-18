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

func FetchGithubFile(owner, repo, branch, path, token string) ([]byte, error) {
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
		return nil, errors.New(fmt.Sprintf("Failed to fetch file:", resp.Status))
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var file GitHubFile
	err = json.Unmarshal(body, &file)

	decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
	return decodedContent, err
}
