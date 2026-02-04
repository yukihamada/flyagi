package github

import (
	"context"
	"fmt"
	"log/slog"

	gh "github.com/google/go-github/v68/github"
)

// Client wraps GitHub API operations.
type Client struct {
	client *gh.Client
	owner  string
	repo   string
}

// NewClient creates a new GitHub API client.
func NewClient(token, owner, repo string) *Client {
	client := gh.NewClient(nil).WithAuthToken(token)
	return &Client{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// CreatePR creates a pull request.
func (c *Client) CreatePR(ctx context.Context, title, body, head, base string) (string, error) {
	pr, _, err := c.client.PullRequests.Create(ctx, c.owner, c.repo, &gh.NewPullRequest{
		Title: gh.Ptr(title),
		Body:  gh.Ptr(body),
		Head:  gh.Ptr(head),
		Base:  gh.Ptr(base),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}

	slog.Info("created pull request", "number", pr.GetNumber(), "url", pr.GetHTMLURL())
	return pr.GetHTMLURL(), nil
}
