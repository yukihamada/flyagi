package git

import (
	"fmt"
	"log/slog"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Service handles git operations.
type Service struct {
	repoPath string
	token    string
	repo     *gogit.Repository
}

// NewService creates a new Git service.
func NewService(repoPath, token string) *Service {
	return &Service{
		repoPath: repoPath,
		token:    token,
	}
}

// CloneOrOpen clones the repository or opens an existing one.
func (s *Service) CloneOrOpen(cloneURL string) error {
	repo, err := gogit.PlainOpen(s.repoPath)
	if err == nil {
		s.repo = repo
		slog.Info("opened existing repository", "path", s.repoPath)
		return nil
	}

	repo, err = gogit.PlainClone(s.repoPath, false, &gogit.CloneOptions{
		URL: cloneURL,
		Auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: s.token,
		},
		Progress: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	s.repo = repo
	slog.Info("cloned repository", "url", cloneURL, "path", s.repoPath)
	return nil
}

// CreateBranch creates a new branch from the current HEAD.
func (s *Service) CreateBranch(name string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	headRef, err := s.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(name)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout the new branch
	wt, err := s.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := wt.Checkout(&gogit.CheckoutOptions{
		Branch: branchRef,
	}); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	slog.Info("created and checked out branch", "name", name)
	return nil
}

// CommitAll stages all changes and commits.
func (s *Service) CommitAll(message string) (string, error) {
	if s.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	wt, err := s.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := wt.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "FlyAGI",
			Email: "flyagi@bot.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	slog.Info("committed changes", "hash", hash.String()[:8], "message", message)
	return hash.String(), nil
}

// Push pushes the current branch to the remote.
func (s *Service) Push(branchName string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	refSpec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))
	err := s.repo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpec},
		Auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: s.token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	slog.Info("pushed branch", "name", branchName)
	return nil
}

// CheckoutMain checks out the main/master branch.
func (s *Service) CheckoutMain() error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	wt, err := s.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try "main" first, then "master"
	for _, branch := range []string{"main", "master"} {
		err = wt.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
		})
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to checkout main branch: %w", err)
}
