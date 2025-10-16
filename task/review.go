package task

import (
	"context"
	"dwight/config"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

type BitbucketPR struct {
	ID          int         `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Comments    []PRComment `json:"comments"`
}

type PRComment struct {
	Text     string `json:"text"`
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
}

func (e *Executor) GetBitbucketReviewComments(cfg *config.Config) (string, error) {
	branch, err := e.getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	pr, err := e.findBitbucketPR(cfg, branch)
	if err != nil {
		return "", fmt.Errorf("failed to find Bitbucket PR: %w", err)
	}

	return e.formatReviewComments(pr), nil
}

func (e *Executor) getCurrentBranch() (string, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

func (e *Executor) findBitbucketPR(cfg *config.Config, branch string) (*BitbucketPR, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, err
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}

	project, repoName, err := e.parseBitbucketRemote(remotes)
	if err != nil {
		return nil, err
	}

	return e.fetchPRDetails(cfg, project, repoName, branch)
}

func (e *Executor) parseBitbucketRemote(remotes []*git.Remote) (string, string, error) {
	for _, remote := range remotes {
		for _, url := range remote.Config().URLs {
			if strings.Contains(url, "bitbucket") {
				parts := strings.Split(url, "/")
				if len(parts) >= 2 {
					project := parts[len(parts)-2]
					repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
					return project, repo, nil
				}
			}
		}
	}
	return "", "", fmt.Errorf("bitbucket remote not found")
}

func (e *Executor) fetchPRDetails(cfg *config.Config, project, repo, branch string) (*BitbucketPR, error) {
	client := NewBitbucketClient(cfg)
	return client.GetPRForBranch(context.Background(), project, repo, branch)
}

func (e *Executor) formatReviewComments(pr *BitbucketPR) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PR #%d: %s\n", pr.ID, pr.Title))

	if pr.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", pr.Description))
	}

	sb.WriteString("\nReview Comments:\n")
	for i, comment := range pr.Comments {
		sb.WriteString(fmt.Sprintf("%d. ", i+1))
		if comment.FilePath != "" {
			sb.WriteString(fmt.Sprintf("%s:%d - ", comment.FilePath, comment.Line))
		}
		sb.WriteString(comment.Text)
		sb.WriteString("\n")
	}

	return sb.String()
}
