package git

import (
	"context"
	"strings"
)

// Branch provides branch management operations.
type Branch struct {
	ops *Ops
}

// NewBranch creates a Branch manager using the given Ops.
func NewBranch(ops *Ops) *Branch {
	return &Branch{ops: ops}
}

// Create creates a new branch from the current HEAD.
func (b *Branch) Create(ctx context.Context, name string) error {
	_, err := b.ops.git(ctx, "branch", name)
	return err
}

// CreateFrom creates a new branch from the given base ref.
func (b *Branch) CreateFrom(ctx context.Context, name, base string) error {
	_, err := b.ops.git(ctx, "branch", name, base)
	return err
}

// Checkout switches to the given branch.
func (b *Branch) Checkout(ctx context.Context, name string) error {
	_, err := b.ops.git(ctx, "checkout", name)
	return err
}

// CheckoutNew creates and switches to a new branch.
func (b *Branch) CheckoutNew(ctx context.Context, name string) error {
	_, err := b.ops.git(ctx, "checkout", "-b", name)
	return err
}

// Delete deletes a branch. If force is true, uses -D instead of -d.
func (b *Branch) Delete(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := b.ops.git(ctx, "branch", flag, name)
	return err
}

// List returns all local branch names.
func (b *Branch) List(ctx context.Context) ([]string, error) {
	out, err := b.ops.git(ctx, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// Current returns the name of the current branch.
func (b *Branch) Current(ctx context.Context) (string, error) {
	return b.ops.CurrentBranch(ctx)
}

// Exists returns whether a branch exists locally.
func (b *Branch) Exists(ctx context.Context, name string) (bool, error) {
	_, err := b.ops.git(ctx, "rev-parse", "--verify", "refs/heads/"+name)
	if err != nil {
		if isCommandError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// isCommandError checks if the error is a git command failure (vs timeout etc).
func isCommandError(err error) bool {
	_, ok := err.(*CommandError)
	return ok
}
