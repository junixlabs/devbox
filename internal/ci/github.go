package ci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitHubProvider implements CIProvider using the GitHub REST API.
type GitHubProvider struct {
	Owner   string
	Repo    string
	Token   string
	BaseURL string // default: https://api.github.com
	Client  *http.Client
}

// NewGitHubProvider creates a new GitHub CI provider.
func NewGitHubProvider(owner, repo, token string) *GitHubProvider {
	return &GitHubProvider{
		Owner:   owner,
		Repo:    repo,
		Token:   token,
		BaseURL: "https://api.github.com",
		Client:  http.DefaultClient,
	}
}

func (g *GitHubProvider) apiURL(path string) string {
	return fmt.Sprintf("%s/repos/%s/%s%s", g.BaseURL, g.Owner, g.Repo, path)
}

func (g *GitHubProvider) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API %s %s returned %d: %s", method, url, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// commentBody builds the PR comment body with the bot marker.
func commentBody(workspaceURL string) string {
	return fmt.Sprintf("%s\n**Devbox Preview**\n\nWorkspace is ready: %s\n\nThis preview will be automatically destroyed when the PR is closed.",
		botCommentMarker, workspaceURL)
}

func (g *GitHubProvider) CommentWorkspaceURL(ctx context.Context, prNumber int, url string) error {
	// Try to find and update existing comment first.
	existingID, err := g.FindBotComment(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("finding existing comment: %w", err)
	}

	body := commentBody(url)

	if existingID != 0 {
		return g.UpdateComment(ctx, existingID, body)
	}

	// Create new comment.
	payload := map[string]string{"body": body}
	apiURL := g.apiURL(fmt.Sprintf("/issues/%d/comments", prNumber))
	_, err = g.doRequest(ctx, http.MethodPost, apiURL, payload)
	if err != nil {
		return fmt.Errorf("creating PR comment: %w", err)
	}
	return nil
}

type ghComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

func (g *GitHubProvider) FindBotComment(ctx context.Context, prNumber int) (int64, error) {
	apiURL := g.apiURL(fmt.Sprintf("/issues/%d/comments?per_page=100", prNumber))
	data, err := g.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("listing PR comments: %w", err)
	}

	var comments []ghComment
	if err := json.Unmarshal(data, &comments); err != nil {
		return 0, fmt.Errorf("parsing comments: %w", err)
	}

	// Return the last matching comment (handles duplicates from race conditions).
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.Contains(comments[i].Body, botCommentMarker) {
			return comments[i].ID, nil
		}
	}
	return 0, nil
}

func (g *GitHubProvider) UpdateComment(ctx context.Context, commentID int64, body string) error {
	apiURL := g.apiURL(fmt.Sprintf("/issues/comments/%d", commentID))
	payload := map[string]string{"body": body}
	_, err := g.doRequest(ctx, http.MethodPatch, apiURL, payload)
	if err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}
	return nil
}

func (g *GitHubProvider) DeleteComment(ctx context.Context, commentID int64) error {
	apiURL := g.apiURL(fmt.Sprintf("/issues/comments/%d", commentID))
	_, err := g.doRequest(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}
	return nil
}

func (g *GitHubProvider) SetCommitStatus(ctx context.Context, sha string, state StatusState, targetURL, description string) error {
	apiURL := g.apiURL(fmt.Sprintf("/statuses/%s", sha))
	payload := map[string]string{
		"state":       string(state),
		"target_url":  targetURL,
		"description": description,
		"context":     "devbox/preview",
	}
	_, err := g.doRequest(ctx, http.MethodPost, apiURL, payload)
	if err != nil {
		return fmt.Errorf("setting commit status: %w", err)
	}
	return nil
}
