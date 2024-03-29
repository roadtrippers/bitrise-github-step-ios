package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

type issue struct {
	Number string
	Title  string
	Labels []string
}

func valueExistsInSlice(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func newRequest(method, url string, body io.Reader) (*http.Request, error) {

	username := os.Getenv("github_username")
	if len(username) == 0 {
		fmt.Println("Error: No username found!")
		os.Exit(1)
	}

	token := os.Getenv("github_personal_access_token")
	if len(token) == 0 {
		fmt.Println("Error: No token found!")
		os.Exit(1)
	}

	req, err := http.NewRequest(method, url, body)
	req.SetBasicAuth(username, token)
	req.Header.Set("Content-Type", "application/json")

	return req, err
}

func main() {

	githubURL := "https://api.github.com"

	buildNumber := os.Getenv("build_number")
	if len(buildNumber) == 0 {
		fmt.Println("Error: No build number found!")
		os.Exit(1)
	}

	fmt.Printf("Using build number %v for Github comments\n", buildNumber)

	version := os.Getenv("version")
	if len(version) == 0 {
		fmt.Println("Error: No version found!")
		os.Exit(1)
	}

	fmt.Printf("Using version %v for Github comments\n", version)

	organization := os.Getenv("github_organization")
	if len(organization) == 0 {
		fmt.Println("Error: No organization found!")
		os.Exit(1)
	}

	fmt.Printf("Using organization %v for requests\n", organization)

	repo := os.Getenv("github_repo")
	if len(repo) == 0 {
		fmt.Println("Error: No repo found!")
		os.Exit(1)
	}

	fmt.Printf("Using repo %v for requests\n", repo)

	var labelsToRemoveSlice []string
	labelsToRemove := os.Getenv("github_labels_to_remove")
	if len(labelsToRemove) > 0 {
		labelsToRemoveSlice = strings.Split(labelsToRemove, ",")
		fmt.Printf("Labels to remove:%v\n", labelsToRemoveSlice)
	}

	var labelsToAddSlice []string
	labelsToAdd := os.Getenv("github_labels_to_add")
	if len(labelsToAdd) > 0 {
		labelsToAddSlice = strings.Split(labelsToAdd, ",")
		fmt.Printf("Labels to add:%v\n", labelsToAddSlice)
	}

	encodedParams := url.PathEscape("q=branch:" + os.Getenv("BITRISE_GIT_BRANCH") + "+in:comments+repo:roadtrippers/roadtrippers-ios+state:open+label:\"needs build\"")
	encodedURL := githubURL + "/search/issues?" + encodedParams
	req, err := newRequest("GET", encodedURL, nil)
	if err != nil {
		fmt.Printf("Error setting up github issue request:%v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error requesting Github issues %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	// Create issue structs
	var issues []issue
	allIssues := gjson.Get(string(body), "items")

	for _, result := range allIssues.Array() {
		var labels []string
		for _, label := range result.Get("labels").Array() {
			labelName := label.Get("name").String()
			if !valueExistsInSlice(labelName, labelsToRemoveSlice) && !valueExistsInSlice(labelName, labelsToAddSlice) {
				labels = append(labels, strconv.Quote(labelName))
			}
		}

		labels = append(labels, strconv.Quote(labelsToAdd))

		fmt.Printf("Labels to update %v\n", labels)

		issue := issue{result.Get("number").String(), result.Get("title").String(), labels}
		issues = append(issues, issue)
	}

	// Construct release notes
	var buf bytes.Buffer
	for _, issue := range issues {
		buf.WriteString(fmt.Sprintf("%s", issue.Title))
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("\n%s - %s", os.Getenv("BITRISE_GIT_BRANCH"), os.Getenv("GIT_CLONE_COMMIT_HASH")))
	releaseNotes := buf.String()

	// Create environment variable for release notes
	c := exec.Command("envman", "add", "--key", "RELEASE_NOTES", "--value", releaseNotes)
	err = c.Run()
	if err != nil {
		fmt.Printf("Error setting RELEASE_NOTES environment variable:%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Release Notes Created:%v\n", releaseNotes)

	githubUsernames := strings.Replace(os.Getenv("github_username_list"), " ", "", -1)
	usernameTags := ""
	if len(githubUsernames) > 0 {
		githubUsernameSlice := strings.Split(githubUsernames, ",")
		for _, username := range githubUsernameSlice {
			usernameTags = usernameTags + "@" + username + " "
		}
		fmt.Printf("Usernames to notify:%v\n", usernameTags)
	} else {
		fmt.Println("No usernames found, not notifying github users.")
	}

	if len(issues) > 0 {
		fmt.Printf("Issues found:%v\n", issues)
		for _, issue := range issues {
			// make labels request
			labelsURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubURL, organization, repo, issue.Number)
			labelsJSONString := []byte(`{"labels":[` + strings.Join(issue.Labels, ",") + `]}`)
			req, err = newRequest("POST", labelsURL, bytes.NewBuffer(labelsJSONString))
			if err != nil {
				fmt.Printf("Error setting up github labels request:%v\n", err)
				os.Exit(1)
			}

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error updating labels:%v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			// make comments request
			commentsURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubURL, organization, repo, issue.Number)
			commentsJSONString := []byte(fmt.Sprintf("{\"body\": \"%s This will be in %s (%s)!\"}", usernameTags, version, buildNumber))
			req, err = newRequest("POST", commentsURL, bytes.NewBuffer(commentsJSONString))
			if err != nil {
				fmt.Printf("Error setting up github comments request:%v\n", err)
				os.Exit(1)
			}

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error updating comments:%v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()
		}
	} else {
		fmt.Println("No issues found!")
	}

	os.Exit(0)
}
