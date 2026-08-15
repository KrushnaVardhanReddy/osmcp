package contracts

// GitAddRequest defines parameters for the git_add tool.
type GitAddRequest struct {
	RepoPath string   `json:"repo_path"`
	Paths    []string `json:"paths"`
}

// GitCommitRequest defines parameters for the git_commit tool.
type GitCommitRequest struct {
	RepoPath    string `json:"repo_path"`
	Message     string `json:"message"`
	AuthorName  string `json:"author_name"`  // Optional
	AuthorEmail string `json:"author_email"` // Optional
}

// GitCheckoutRequest defines parameters for the git_checkout tool.
type GitCheckoutRequest struct {
	RepoPath string `json:"repo_path"`
	Branch   string `json:"branch"`
	Create   bool   `json:"create"`
}

// GitBranchRequest defines parameters for the git_branch tool.
type GitBranchRequest struct {
	RepoPath   string `json:"repo_path"`
	Action     string `json:"action"` // "create" or "delete"
	BranchName string `json:"branch_name"`
}

// GitPullRequest defines parameters for the git_pull tool.
type GitPullRequest struct {
	RepoPath string `json:"repo_path"`
	Remote   string `json:"remote"` // Default: "origin"
	Branch   string `json:"branch"`
}

// GitPushRequest defines parameters for the git_push tool.
type GitPushRequest struct {
	RepoPath string `json:"repo_path"`
	Remote   string `json:"remote"` // Default: "origin"
	Branch   string `json:"branch"`
	Force    bool   `json:"force"`  // Default: false
}

// GitMutationTool implementations must embed the policy engine and response builder.
type GitMutationTool interface {
	GitAdd(req GitAddRequest) Envelope
	GitCommit(req GitCommitRequest) Envelope
	GitCheckout(req GitCheckoutRequest) Envelope
	GitBranch(req GitBranchRequest) Envelope
	GitPull(req GitPullRequest) Envelope
	GitPush(req GitPushRequest) Envelope
}
