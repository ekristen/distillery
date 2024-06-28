package source

import (
	"context"
)

type GoReleaser struct {
	DataFile      string
	SignatureFile string
	KeyFile       string
	ChecksumFile  string
}

func (g *GoReleaser) Verify(ctx context.Context) error {
	return nil
}
