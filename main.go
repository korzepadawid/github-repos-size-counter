package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type GithubRepo struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

func main() {
	fmt.Println(countGithubReposSizeByUserLogin("facebook"))
}

func countGithubReposSizeByUserLogin(username string) (int, error) {
	resp, err := sendGithubReposRequest(username, 1)

	if err != nil {
		return 0, err
	}

	githubRepos := parseResponseBody(resp)
	size := countGithubReposSizeFromSinglePage(githubRepos)
	pages := getGithubReposPageCount(resp)

	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	for i := 2; i <= pages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			res, _ := sendGithubReposRequest(username, page)
			githubRepos = parseResponseBody(res)
			tempSize := countGithubReposSizeFromSinglePage(githubRepos)
			mutex.Lock()
			size += tempSize
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	return size, nil
}

func sendGithubReposRequest(username string, pageNumber int) (*http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?page=%d", username, pageNumber)
	resp, err := http.Get(url)
	return resp, err
}

func getGithubReposPageCount(r *http.Response) int {
	linkHeader := r.Header.Get("Link")

	if linkHeader == "" {
		return 1
	}

	re := regexp.MustCompile("<(.*)>; *rel=\"(.*)\"")

	splittedLinks := strings.Split(linkHeader, ",")

	for _, link := range splittedLinks {
		match := re.FindStringSubmatch(link)
		urlToParse := match[1]
		rel := match[2]

		if rel == "last" {
			parsedUrl, _ := url.Parse(urlToParse)
			pageNumber, _ := strconv.Atoi(parsedUrl.Query().Get("page"))
			return pageNumber
		}
	}

	return 1
}

func parseResponseBody(r *http.Response) *[]GithubRepo {
	body := &[]GithubRepo{}
	json.NewDecoder(r.Body).Decode(body)
	return body
}

func countGithubReposSizeFromSinglePage(repos *[]GithubRepo) int {
	var size int
	for _, repo := range *repos {
		size += repo.Size
	}
	return size
}
