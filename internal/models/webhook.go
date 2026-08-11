package models

// Repository represents repository details in GitHub webhook payloads.
type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// Sender represents user details who triggered the webhook event.
type Sender struct {
	Login string `json:"login"`
}

// HeadCommit represents commit details for push events.
type HeadCommit struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// PushPayload represents GitHub push webhook event payload.
type PushPayload struct {
	Repository Repository `json:"repository"`
	Sender     Sender     `json:"sender"`
	Ref        string     `json:"ref"`
	HeadCommit HeadCommit `json:"head_commit"`
}

// PullRequestDetails represents pull request details.
type PullRequestDetails struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// PullRequestPayload represents GitHub pull_request webhook event payload.
type PullRequestPayload struct {
	Action      string             `json:"action"`
	Repository  Repository         `json:"repository"`
	PullRequest PullRequestDetails `json:"pull_request"`
	Sender      Sender             `json:"sender"`
}

// HealthResponse represents the health check status response.
type HealthResponse struct {
	Status string `json:"status"`
}
