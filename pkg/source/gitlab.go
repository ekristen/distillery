package source //nolint:dupl

import (
	"context"
	"fmt"
)

type GitLab struct {
	Source

	Owner   string
	Repo    string
	Version string
}

func (g *GitLab) GetSource() string {
	return "gitlab"
}
func (g *GitLab) GetOwner() string {
	return g.Owner
}
func (g *GitLab) GetRepo() string {
	return g.Repo
}
func (g *GitLab) GetApp() string {
	return fmt.Sprintf("%s/%s", g.Owner, g.Repo)
}
func (g *GitLab) GetID() string {
	return fmt.Sprintf("%s/%s/%s", g.GetSource(), g.GetOwner(), g.GetRepo())
}
func (g *GitLab) Run(_ context.Context, _, _ string) error {
	return nil
}
