package source

import "context"

type Homebrew struct {
	Source

	Formula string
	Version string
}

func (s *Homebrew) GetSource() string {
	return "homebrew"
}
func (s *Homebrew) GetOwner() string {
	return ""
}
func (s *Homebrew) GetRepo() string {
	return s.Formula
}
func (s *Homebrew) GetApp() string {
	return s.Formula
}
func (s *Homebrew) GetID() string {
	return s.Formula
}
func (s *Homebrew) Run(_ context.Context, _, _ string) error {
	return nil
}
