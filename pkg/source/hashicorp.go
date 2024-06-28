package source //nolint:dupl

import (
	"context"
	"fmt"
)

type Hashicorp struct {
	Source

	Owner   string
	Repo    string
	Version string
}

func (s *Hashicorp) GetSource() string {
	return "hashicorp"
}
func (s *Hashicorp) GetOwner() string {
	return s.Owner
}
func (s *Hashicorp) GetRepo() string {
	return s.Repo
}
func (s *Hashicorp) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}
func (s *Hashicorp) GetID() string {
	return fmt.Sprintf("%s/%s/%s", s.GetSource(), s.GetOwner(), s.GetRepo())
}
func (s *Hashicorp) Run(_ context.Context, _, _ string) error {
	return nil
}
