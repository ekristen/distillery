package asset

import "context"

type IAsset interface {
	GetName() string
	GetDisplayName() string
	GetType() Type
	GetParentType() Type
	GetAsset() *Asset
	GetFiles() []*File
	GetTempPath() string
	GetFilePath() string
	Download(context.Context) error
	Extract() error
	Install(string, string, string) error
	Cleanup() error
	ID() string
	Path() string
	GetChecksumType() string
	GetMatchedAsset() IAsset
	SetMatchedAsset(IAsset)
	GetGPGKeyID() (uint64, error)
	GetBaseName() string
}
