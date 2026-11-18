package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/tidwall/gjson"
)

var githubURL = "https://api.github.com"

type issue struct {
	Number int64
	Title  string
}

// dispatchPayload is sent as client_payload on the repository_dispatch event.
// The GitHub Actions workflow in the target repo uses it to comment on and
// relabel the issues. GitHub allows at most 10 top-level keys here.
type dispatchPayload struct {
	Version        string   `json:"version"`
	BuildNumber    string   `json:"build_number"`
	Branch         string   `json:"branch"`
	Commit         string   `json:"commit"`
	Issues         []int64  `json:"issues"`
	LabelsToAdd    []string `json:"labels_to_add"`
	LabelsToRemove []string `json:"labels_to_remove"`
	Usernames      []string `json:"usernames"`
}

func fail(format string, a ...interface{}) {
	fmt.Printf("Error: "+format+"\n", a...)
	os.Exit(1)
}

func requireEnv(key, name string) string {
	value := os.Getenv(key)
	if value == "" {
		fail("No %s found!", name)
	}
	fmt.Printf("Using %s %v\n", name, value)
	return value
}

// splitList turns a comma separated input into a trimmed slice, dropping empties.
func splitList(value string) []string {
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// githubToken prefers the short-lived installation token that the Bitrise
// GitHub App integration injects into every build. GITHUB_PAT is only a
// fallback while projects migrate and should be removed afterwards.
func githubToken() string {
	if token := os.Getenv("GIT_HTTP_PASSWORD"); token != "" {
		fmt.Println("Authenticating with the Bitrise GitHub App token (GIT_HTTP_PASSWORD)")
		return token
	}
	if token := os.Getenv("GITHUB_PAT"); token != "" {
		fmt.Println("Warning: GIT_HTTP_PASSWORD not set, falling back to GITHUB_PAT. Remove the PAT once the GitHub App is set up.")
		return token
	}
	fail("Neither GIT_HTTP_PASSWORD (Bitrise GitHub App) nor GITHUB_PAT is set.")
	return ""
}

// githubRequest performs an authenticated request and returns the body,
// exiting on transport errors or non-2xx responses.
func githubRequest(token, method, requestURL string, body []byte, action string) []byte {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, requestURL, reader)
	if err != nil {
		fail("setting up request to %s: %v", action, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail("requesting %s: %v", action, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("reading response to %s: %v", action, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		hint := ""
		if resp.StatusCode == 403 || resp.StatusCode == 404 {
			hint = " (check that \"Extend GitHub App permissions to builds\" is enabled in the Bitrise project's Repository settings)"
		}
		fail("%s: HTTP %d - %s%s", action, resp.StatusCode, string(respBody), hint)
	}

	return respBody
}

func main() {
	if override := os.Getenv("GITHUB_API_URL"); override != "" {
		githubURL = strings.TrimSuffix(override, "/")
	}

	buildNumber := requireEnv("build_number", "build number")
	version := requireEnv("version", "version")
	organization := requireEnv("github_organization", "organization")
	repo := requireEnv("github_repo", "repo")

	eventType := os.Getenv("github_dispatch_event_type")
	if eventType == "" {
		eventType = "bitrise-build-ready"
	}

	branch := os.Getenv("BITRISE_GIT_BRANCH")
	commit := os.Getenv("GIT_CLONE_COMMIT_HASH")
	token := githubToken()

	// Find the open issues waiting on this branch's build.
	query := url.PathEscape(fmt.Sprintf("q=is:issue+branch:%s+in:comments+repo:%s/%s+state:open+label:\"needs build\"", branch, organization, repo))
	searchBody := githubRequest(token, "GET", githubURL+"/search/issues?"+query, nil, "search GitHub issues")

	var issues []issue
	for _, result := range gjson.GetBytes(searchBody, "items").Array() {
		issues = append(issues, issue{result.Get("number").Int(), result.Get("title").String()})
	}

	// Build release notes from the issue titles.
	var buf bytes.Buffer
	for _, issue := range issues {
		buf.WriteString(issue.Title)
		buf.WriteString("\n")
	}
	buf.WriteString(fmt.Sprintf("\n%s - %s", branch, commit))
	releaseNotes := buf.String()

	if err := exec.Command("envman", "add", "--key", "RELEASE_NOTES", "--value", releaseNotes).Run(); err != nil {
		// Allow skipping release notes so the test script can run outside Bitrise.
		fmt.Printf("Error setting RELEASE_NOTES environment variable:%v\n", err)
		fmt.Println("Skipping Release Notes")
	} else {
		fmt.Printf("Release Notes Created:%v\n", releaseNotes)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found, nothing to dispatch.")
		os.Exit(0)
	}

	// Hand the write work (comments + labels) to GitHub Actions in the target
	// repo, which uses its own GITHUB_TOKEN. The Bitrise token can only read issues.
	payload := dispatchPayload{
		Version:        version,
		BuildNumber:    buildNumber,
		Branch:         branch,
		Commit:         commit,
		LabelsToAdd:    splitList(os.Getenv("github_labels_to_add")),
		LabelsToRemove: splitList(os.Getenv("github_labels_to_remove")),
		Usernames:      splitList(os.Getenv("github_username_list")),
	}
	for _, issue := range issues {
		payload.Issues = append(payload.Issues, issue.Number)
	}

	body, err := json.Marshal(map[string]interface{}{
		"event_type":     eventType,
		"client_payload": payload,
	})
	if err != nil {
		fail("encoding dispatch payload: %v", err)
	}

	fmt.Printf("Sending %s dispatch to %s/%s: %s\n", eventType, organization, repo, string(body))
	githubRequest(token, "POST", fmt.Sprintf("%s/repos/%s/%s/dispatches", githubURL, organization, repo), body, "send repository_dispatch")
	fmt.Printf("Dispatched %d issue(s) to GitHub Actions for comments and labels.\n", len(issues))

	os.Exit(0)
}
