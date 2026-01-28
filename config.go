package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/codingconcepts/env"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logLevels = []string{"debug", "info", "warn", "error"}

type LogLevel string

func (v LogLevel) String() string {
	return string(v)
}

func (v LogLevel) Validate() error {
	if v == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	if !slices.Contains(logLevels, string(v)) {
		return fmt.Errorf("invalid log level: %s (allowed: %s)", v, strings.Join(logLevels, ", "))
	}

	return nil
}

func (v *LogLevel) Set(raw string) error {
	*v = LogLevel(raw)
	return v.Validate()
}

var logFormats = []string{"text", "json"}

type LogFormat string

func (v LogFormat) String() string {
	return string(v)
}

func (v LogFormat) Validate() error {
	if v == "" {
		return fmt.Errorf("log format cannot be empty")
	}

	if !slices.Contains(logFormats, string(v)) {
		return fmt.Errorf("invalid log format: %s (allowed: %s)", v, strings.Join(logFormats, ", "))
	}

	return nil
}

func (v *LogFormat) Set(raw string) error {
	*v = LogFormat(raw)
	return v.Validate()
}

type Redirect struct {
	From   string
	To     string
	Status int
}

type Redirects []Redirect

func (r Redirects) String() string {
	parts := make([]string, 0, len(r))
	for _, redirect := range r {
		parts = append(parts, fmt.Sprintf("%s=>%s=%d", redirect.From, redirect.To, redirect.Status))
	}

	return strings.Join(parts, "; ")
}

func (r Redirects) Has(from string) bool {
	for _, redirect := range r {
		if redirect.From == from {
			return true
		}
	}

	return false
}

func (r Redirects) Validate() error {
	seen := make(map[string]struct{})
	for _, redirect := range r {
		if _, exists := seen[redirect.From]; exists {
			return fmt.Errorf("duplicate redirect from %s", redirect.From)
		}
		seen[redirect.From] = struct{}{}

		if redirect.Status < 200 || redirect.Status > 399 {
			return fmt.Errorf("invalid status code %d for redirect from %s", redirect.Status, redirect.From)
		}
	}

	return nil
}

func (r *Redirects) Set(value string) error {
	var errInvalidRedirectFormat = fmt.Errorf("invalid redirect format (expected from=>to=status[;...])")

	items := strings.Split(value, ";")
	for _, item := range items {
		if item == "" {
			continue
		}

		item = strings.TrimSpace(item)
		parts := strings.Split(item, "=>")
		if len(parts) != 2 {
			return fmt.Errorf("%w: %s", errInvalidRedirectFormat, item)
		}

		var (
			from      = parts[0]
			to        = parts[1]
			statusStr string
			status    = 301
			found     bool
			err       error
		)

		if to, statusStr, found = strings.Cut(to, "="); found {
			if status, err = strconv.Atoi(statusStr); err != nil {
				return fmt.Errorf("%w: failed to parse status code %s: %w", errInvalidRedirectFormat, statusStr, err)
			}
		}

		(*r) = append((*r), Redirect{
			From:   from,
			To:     to,
			Status: status,
		})
	}

	return r.Validate()
}

func (r *Redirects) Merge(other Redirects) {
	for _, redirect := range other {
		if !r.Has(redirect.From) {
			*r = append(*r, redirect)
		}
	}
}

type Config struct {
	LogLevel      LogLevel  `env:"LOG_LEVEL" default:"info"`
	LogFormat     LogFormat `env:"LOG_FORMAT" default:"json"`
	ServerAddress string    `env:"SERVER_ADDRESS" default:":8000"`
	Redirects     Redirects
}

func (c *Config) Validate() error {
	if err := c.LogLevel.Validate(); err != nil {
		return err
	}

	if err := c.LogFormat.Validate(); err != nil {
		return err
	}

	if err := c.Redirects.Validate(); err != nil {
		return err
	}

	return nil
}

var config Config

func init() {
	if err := env.Set(&config); err != nil {
		panic(fmt.Errorf("failed to set config from environment variables: %w", err))
	}

	flag.Var(&config.LogFormat, "log-format", "Log format (text|json)")
	flag.Var(&config.LogLevel, "log-level", "Log level (debug|info|warn|error)")
	flag.Var(&config.Redirects, "redirects", "List of redirects in the format from0=>to0=status; from1=>to1=status; ...")
	flag.Parse()

	if err := config.Validate(); err != nil {
		panic(fmt.Errorf("invalid configuration: %w", err))
	}

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	level, err := zerolog.ParseLevel(config.LogLevel.String())
	if err != nil {
		panic(fmt.Errorf("failed to parse log level %s: %w", config.LogLevel, err))
	}
	zerolog.SetGlobalLevel(level)
	switch config.LogFormat.String() {
	case "json":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
		log.Logger = log.Logger.
			With().Caller().Logger()
	case "text":
		log.Logger = zerolog.
			New(zerolog.NewConsoleWriter()).
			With().Timestamp().Caller().Logger()
	}
}
