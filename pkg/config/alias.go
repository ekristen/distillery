package config

import (
	"strings"

	"github.com/ekristen/distillery/pkg/common"
)

type Aliases map[string]*Alias

type Alias struct {
	Name    string          `yaml:"name" toml:"name"`
	Version string          `yaml:"version" toml:"version"`
	Flags   map[string]bool `yaml:"flags" toml:"flags"`
}

// DefaultAliases - default aliases for distillery, this will only ever be `dist`, I have no plans on maintain
// aliases for other projects, that is what the configuration is for and is part of the design of this tool, no
// central repository.
var DefaultAliases = Aliases{
	"dist": {
		Name:    "github/ekristen/distillery",
		Version: "latest",
	},
}

func (a *Alias) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if unmarshal(&value) == nil {
		p := strings.Split(value, "@")
		a.Name = p[0]
		a.Version = common.Latest
		if len(p) > 1 {
			a.Version = p[1]
		}
		return nil
	}

	type alias Alias
	aux := (*alias)(a)
	if err := unmarshal(aux); err != nil {
		return err
	}

	return nil
}

func (a *Alias) UnmarshalText(b []byte) error {
	*a = Alias{
		Name:    string(b),
		Version: "latest",
	}
	return nil
}
