package provider

import (
	"context"
)

type ISource interface {
	GetSource() string
	GetOwner() string
	GetRepo() string
	GetApp() string
	GetID() string
	GetDownloadsDir() string
	GetVersion() string
	PreRun(context.Context) error
	Run(context.Context) error
	GetOptions() *Options
}
