// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PRBranches holds the source and target branch names of a pull request.
type PRBranches struct {
	Source string
	Target string
}

// Client interacts with Bitbucket Server REST API v1.0.
//
//counterfeiter:generate -o ../../mocks/bitbucket-client.go --fake-name BitbucketClient . Client
type Client interface {
	GetPRBranches(ctx context.Context, host, project, repo string, number int) (PRBranches, error)
	PostComment(ctx context.Context, host, project, repo string, number int, body string) error
	Approve(ctx context.Context, host, project, repo string, number int) error
	NeedsWork(ctx context.Context, host, project, repo string, number int, userSlug string) error
}

// NewClient creates a Client that uses the Bitbucket Server REST API.
// Token is used for Bearer authentication.
func NewClient(token string) Client {
	return &httpClient{
		token:      token,
		httpClient: &http.Client{},
	}
}

type httpClient struct {
	token      string
	httpClient *http.Client
}

type prResponse struct {
	FromRef struct {
		DisplayID string `json:"displayId"`
	} `json:"fromRef"`
	ToRef struct {
		DisplayID string `json:"displayId"`
	} `json:"toRef"`
}

type commentRequest struct {
	Text string `json:"text"`
}

type participantRequest struct {
	User     participantUser `json:"user"`
	Approved bool            `json:"approved"`
	Status   string          `json:"status"`
}

type participantUser struct {
	Slug string `json:"slug"`
}

// GetPRBranches fetches the source and target branch names for a pull request.
func (c *httpClient) GetPRBranches(
	ctx context.Context,
	host, project, repo string,
	number int,
) (PRBranches, error) {
	url := c.buildURL(host, fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d",
		project, repo, number))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PRBranches{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PRBranches{}, fmt.Errorf("request failed for %s: %w", host, err)
	}
	defer resp.Body.Close()

	if err := checkResponseStatus(resp, host, project, repo, number); err != nil {
		return PRBranches{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PRBranches{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var prResp prResponse
	if err := json.Unmarshal(body, &prResp); err != nil {
		return PRBranches{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if prResp.FromRef.DisplayID == "" {
		return PRBranches{}, fmt.Errorf("PR response missing source branch")
	}
	if prResp.ToRef.DisplayID == "" {
		return PRBranches{}, fmt.Errorf("PR response missing target branch")
	}

	return PRBranches{
		Source: prResp.FromRef.DisplayID,
		Target: prResp.ToRef.DisplayID,
	}, nil
}

// PostComment posts a comment on a pull request.
func (c *httpClient) PostComment(
	ctx context.Context,
	host, project, repo string,
	number int,
	body string,
) error {
	url := c.buildURL(
		host,
		fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments",
			project, repo, number),
	)

	commentReq := commentRequest{Text: body}
	jsonData, err := json.Marshal(commentReq)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", host, err)
	}
	defer resp.Body.Close()

	if err := checkResponseStatus(resp, host, project, repo, number); err != nil {
		return err
	}

	return nil
}

// Approve approves a pull request.
func (c *httpClient) Approve(
	ctx context.Context,
	host, project, repo string,
	number int,
) error {
	url := c.buildURL(
		host,
		fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/approve",
			project, repo, number),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", host, err)
	}
	defer resp.Body.Close()

	if err := checkApproveResponseStatus(resp, host, project, repo, number); err != nil {
		return err
	}

	return nil
}

// NeedsWork marks a pull request as needing work.
func (c *httpClient) NeedsWork(
	ctx context.Context,
	host, project, repo string,
	number int,
	userSlug string,
) error {
	url := c.buildURL(
		host,
		fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants/%s",
			project, repo, number, userSlug),
	)

	participantReq := participantRequest{
		User: participantUser{
			Slug: userSlug,
		},
		Approved: false,
		Status:   "NEEDS_WORK",
	}

	jsonData, err := json.Marshal(participantReq)
	if err != nil {
		return fmt.Errorf("failed to marshal participant request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", host, err)
	}
	defer resp.Body.Close()

	if err := checkResponseStatus(resp, host, project, repo, number); err != nil {
		return err
	}

	return nil
}

// buildURL constructs the full URL with scheme detection.
// If host contains a scheme (http:// or https://), use it as-is.
// Otherwise, default to https:// for production Bitbucket Server instances.
func (c *httpClient) buildURL(host, path string) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + path
	}
	return "https://" + host + path
}

// checkResponseStatus validates HTTP response status and returns appropriate errors.
// Token is intentionally excluded from error messages for security.
func checkResponseStatus(resp *http.Response, host, project, repo string, number int) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed for %s", host)
	case http.StatusForbidden:
		return fmt.Errorf("insufficient permissions for %s", host)
	case http.StatusNotFound:
		return fmt.Errorf("PR not found: %s/projects/%s/repos/%s/pull-requests/%d",
			host, project, repo, number)
	default:
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, host)
	}
}

// checkApproveResponseStatus validates HTTP response status for approve requests.
// Treats 409 Conflict (already approved) as success.
// Token is intentionally excluded from error messages for security.
func checkApproveResponseStatus(resp *http.Response, host, project, repo string, number int) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusConflict:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed for %s", host)
	case http.StatusForbidden:
		return fmt.Errorf("insufficient permissions for %s", host)
	case http.StatusNotFound:
		return fmt.Errorf("PR not found: %s/projects/%s/repos/%s/pull-requests/%d",
			host, project, repo, number)
	default:
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, host)
	}
}
