package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HostReplies          []HostReplyConfig          `yaml:"host_replies,omitempty"`
	NoqueueRejectReplies []NoqueueRejectReplyConfig `yaml:"noqueue_reject_replies,omitempty"`
}

func Load(name string) (*Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	var cfg Config
	if err = d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}
	return &cfg, nil
}

type HostReplyConfig struct {
	Type   HostReplyType `yaml:"type,omitempty"`
	Regexp *Regexp       `yaml:"regexp"`
	Text   string        `yaml:"text"`
}

func (cfg *HostReplyConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain HostReplyConfig
	if err := value.Decode((*plain)(cfg)); err != nil {
		return err
	}
	if cfg.Text == "" {
		return errors.New("empty text replacement")
	}
	return nil
}

type HostReplyType int

func (t *HostReplyType) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch s {
	case "", "any":
		*t = HostReplyAny
	case "queue_status":
		*t = HostReplyQueueStatus
	case "other":
		*t = HostReplyOther
	default:
		return fmt.Errorf("unsupported host reply type %q", s)
	}
	return nil
}

// HostReplyType types.
const (
	HostReplyAny HostReplyType = iota
	HostReplyQueueStatus
	HostReplyOther
)

type NoqueueRejectReplyConfig struct {
	Regexp *Regexp `yaml:"regexp"`
	Text   string  `yaml:"text"`
}

func (cfg *NoqueueRejectReplyConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain NoqueueRejectReplyConfig
	if err := value.Decode((*plain)(cfg)); err != nil {
		return err
	}
	if cfg.Text == "" {
		return errors.New("empty text replacement")
	}
	return nil
}

type Regexp struct {
	*regexp.Regexp
}

func (r *Regexp) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return err
	}
	*r = Regexp{re}
	return nil
}
