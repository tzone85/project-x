package git

import "time"

// DefaultTimeout is the default command timeout.
const DefaultTimeout = 30 * time.Second

// StatusEntry represents a single file's git status.
type StatusEntry struct {
	Staging  byte
	Worktree byte
	Path     string
}

// LogEntry represents a single git log entry.
type LogEntry struct {
	Hash    string
	Author  string
	Date    string
	Subject string
}

// DiffStat summarizes a diff operation.
type DiffStat struct {
	FilesChanged int
	Insertions   int
	Deletions    int
	Raw          string
}

// WorktreeInfo describes a git worktree.
type WorktreeInfo struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}

// PRInfo describes a GitHub pull request.
type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
	Branch string `json:"headRefName"`
	Base   string `json:"baseRefName"`
	Author string `json:"author"`
}

// PRCreateOptions configures PR creation.
type PRCreateOptions struct {
	Title string
	Body  string
	Base  string
	Head  string
	Draft bool
}

// PRMergeOptions configures PR merging.
type PRMergeOptions struct {
	Method        string // merge, squash, rebase
	DeleteBranch  bool
	AutoMerge     bool
	CommitSubject string
}

// TechStack holds detected technology information for a repository.
type TechStack struct {
	Languages       []Language
	Frameworks      []string
	TestRunners     []string
	Linters         []string
	BuildTools      []string
	PackageManagers []string
	DirectoryLayout string
}

// Language describes a detected programming language.
type Language struct {
	Name    string
	Version string
	Primary bool
}

// ConflictInfo describes a merge/rebase conflict.
type ConflictInfo struct {
	InProgress    bool
	Type          string // "merge", "rebase", "cherry-pick", or ""
	ConflictFiles []string
}
