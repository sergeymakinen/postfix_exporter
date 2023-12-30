package config

import (
	"errors"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	StatusReplies        []StatusReplyMatchConfig `yaml:"status_replies,omitempty"`
	SmtpReplies          []ReplyMatchConfig       `yaml:"smtp_replies,omitempty"`
	NoqueueRejectReplies []ReplyMatchConfig       `yaml:"noqueue_reject_replies,omitempty"`
}

func Load(name string) (*Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.New("error reading config file: " + err.Error())
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	var cfg Config
	if err = d.Decode(&cfg); err != nil {
		return nil, errors.New("error parsing config file: " + err.Error())
	}
	return &cfg, nil
}

type StatusReplyMatchConfig struct {
	Statuses    []string  `yaml:"statuses,omitempty"`
	NotStatuses []string  `yaml:"not_statuses,omitempty"`
	Regexp      *Regexp   `yaml:"regexp"`
	Match       MatchType `yaml:"match,omitempty"`
	Text        string    `yaml:"text"`
}

func (cfg *StatusReplyMatchConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain StatusReplyMatchConfig
	if err := value.Decode((*plain)(cfg)); err != nil {
		return err
	}
	if cfg.Text == "" {
		return errors.New("empty text replacement")
	}
	return nil
}

type ReplyMatchConfig struct {
	Regexp *Regexp   `yaml:"regexp"`
	Match  MatchType `yaml:"match,omitempty"`
	Text   string    `yaml:"text"`
}

func (cfg *ReplyMatchConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain ReplyMatchConfig
	if err := value.Decode((*plain)(cfg)); err != nil {
		return err
	}
	if cfg.Text == "" {
		return errors.New("empty text replacement")
	}
	return nil
}

type MatchType int

func (t *MatchType) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch s {
	case "", "text":
		*t = MatchTypeText
	case "code":
		*t = MatchTypeCode
	case "enhanced_code":
		*t = MatchTypeEnhancedCode
	default:
		return errors.New("unsupported match type " + strconv.Quote(s))
	}
	return nil
}

// MatchType types.
const (
	MatchTypeText MatchType = iota
	MatchTypeCode
	MatchTypeEnhancedCode
)

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
