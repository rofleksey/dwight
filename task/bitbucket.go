package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rofleksey/dwight/config"
)

type BitbucketClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewBitbucketClient(cfg *config.Config) *BitbucketClient {
	return &BitbucketClient{
		baseURL: cfg.BitbucketHost,
		token:   cfg.BitbucketToken,
		client:  &http.Client{},
	}
}

func (b *BitbucketClient) GetPRForBranch(ctx context.Context, project, repo, branch string) (*BitbucketPR, error) {
	prs, err := b.listPRs(ctx, project, repo)
	if err != nil {
		return nil, err
	}

	for _, pr := range prs {
		if strings.Contains(pr.SourceRef, branch) {
			return b.getPRDetails(ctx, project, repo, pr.ID)
		}
	}

	return nil, fmt.Errorf("no PR found for branch %s", branch)
}

func (b *BitbucketClient) listPRs(ctx context.Context, project, repo string) ([]bitbucketPRSummary, error) {
	endpoint := fmt.Sprintf("%s/rest/api/latest/projects/%s/repos/%s/pull-requests",
		b.baseURL, project, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.token)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bitbucket API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Values []bitbucketPRSummary `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

func (b *BitbucketClient) getPRDetails(ctx context.Context, project, repo string, prID int) (*BitbucketPR, error) {
	endpoint := fmt.Sprintf("%s/rest/api/latest/projects/%s/repos/%s/pull-requests/%d",
		b.baseURL, project, repo, prID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.token)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bitbucket API returned status: %d", resp.StatusCode)
	}

	var pr BitbucketPR
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}

	comments, err := b.getPRComments(ctx, project, repo, prID)
	if err != nil {
		return nil, err
	}
	pr.Comments = comments

	return &pr, nil
}

func (b *BitbucketClient) getPRComments(ctx context.Context, project, repo string, prID int) ([]PRComment, error) {
	endpoint := fmt.Sprintf("%s/rest/api/latest/projects/%s/repos/%s/pull-requests/%d/activities",
		b.baseURL, project, repo, prID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.token)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bitbucket API returned status: %d: %s", resp.StatusCode, string(body))
	}

	var activities []bitbucketActivity
	if err := json.Unmarshal(body, &activities); err != nil {
		return nil, err
	}

	var comments []PRComment
	for _, activity := range activities {
		if activity.Comment != nil {
			comment := PRComment{
				Text:     activity.Comment.Text,
				FilePath: activity.Comment.Anchor.Path,
				Line:     activity.Comment.Anchor.Line,
			}
			comments = append(comments, comment)
		}
	}

	return comments, nil
}

type bitbucketPRSummary struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	SourceRef string `json:"fromRef"`
}

type bitbucketActivity struct {
	Comment *struct {
		Text   string `json:"text"`
		Anchor struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"anchor"`
	} `json:"comment"`
}
