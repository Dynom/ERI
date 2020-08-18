package config

import (
	"encoding"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	VTStructure ValidatorType = "structure"
	VTLookup    ValidatorType = "lookup"

	LFJSON LogFormat = "json"
	LGText LogFormat = "text"
)

func NewConfig(fileName string) (Config, error) {
	c := Config{}

	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return c, fmt.Errorf("unable to open %q, reason: %w", fileName, err)
	}

	_, err = toml.Decode(string(b), &c)
	if err != nil {
		return c, fmt.Errorf("unable to unmarshal %q, reason: %w", fileName, err)
	}

	return c, nil
}

// Config holds central config parameters
type Config struct {
	Server struct {
		ListenOn        string   `toml:"listenOn"`
		ConnectionLimit uint     `toml:"connectionLimit"`
		InstanceID      string   `toml:"-"` // Extra identifier used in logs and for instance identification
		MaxRequestSize  uint64   `toml:"maxRequestSize" usage:"Maximum amount of bytes until a HTTP request is accepted"`
		NetTTL          Duration `toml:"netTTL" usage:"Max time to spend on external communication"`
		PathStrip       string   `toml:"pathStrip"`
		Headers         Headers  `toml:"headers" env:"-" usage:"Only (repeatable) flag or config file supported"`
		CORS            struct {
			AllowedOrigins []string `toml:"allowedOrigins"`
			AllowedHeaders []string `toml:"allowedHeaders"`
		} `toml:"CORS"`
		Profiler struct {
			Enable bool   `toml:"enable" default:"false"`
			Prefix string `toml:"prefix"`
		} `toml:"profiler"`
	} `toml:"server" flag:",inline" env:",inline"`
	Log struct {
		Level  string    `toml:"level"`
		Format LogFormat `toml:"format" usage:"The log output format \"json\" or \"text\""`
	} `toml:"log"`
	Hash struct {
		Key string `toml:"key"`
	} `toml:"hash"`
	Finder struct {
		UseBuckets      bool    `toml:"useBuckets" usage:"Buckets speedup matching, but assumes no mistakes are made at the start"`
		LengthTolerance float64 `toml:"lengthTolerance" usage:"percentage, number 0.0-1.0, of length difference to consider"`
	} `toml:"finder"`
	Validator struct {
		Resolver         string        `toml:"resolver" usage:"The resolver to use for DNS lookups"`
		SuggestValidator ValidatorType `toml:"suggest"`
	} `toml:"validator" flag:",inline" env:",inline"`
	Services struct {
		Autocomplete struct {
			RecipientThreshold uint64 `toml:"recipientThreshold" usage:"Define the minimum amount of recipients a domain needs before allowed in the autocomplete"`
			MaxSuggestions     uint64 `toml:"maxSuggestions" usage:"The maximum number of suggestions to return"`
		} `toml:"autocomplete"`
		Suggest struct {
			Prefer Preferred `toml:"prefer" env:"-" usage:"A repeatable flag to create a preference list for common alternatives, example.com=example.org"`
		} `toml:"suggest"`
	} `toml:"services"`
	Backend struct {
		Driver             string `toml:"driver" usage:"List a driver to use, currently supporting: 'memory' or 'postgres'"`
		URL                string `toml:"url"`
		MaxConnections     uint   `toml:"maxConnections"`
		MaxIdleConnections uint   `toml:"maxIdleConnections"`
	} `toml:"backend"`
	GraphQL struct {
		PrettyOutput bool `toml:"prettyOutput" flag:"pretty" env:"PRETTY"`
		GraphiQL     bool `toml:"graphiQL" flag:"graphiql" env:"GRAPHIQL"`
		Playground   bool `toml:"playground"`
	} `toml:"graphql" flag:"graphql" env:"GRAPHQL"`
	RateLimiter struct {
		Rate      int64    `toml:"rate"`
		Capacity  int64    `toml:"capacity"`
		ParkedTTL Duration `toml:"parkedTTL" flag:"parked-ttl"`
	} `toml:"rateLimiter"`
	GCP struct {
		ProjectID       string `toml:"projectId"`
		PubSubTopic     string `toml:"pubSubTopic"`
		CredentialsFile string `toml:"credentialsFile" env:"APPLICATION_CREDENTIALS"`
	} `toml:"GCP" flag:"gcp" env:"GOOGLE"`
}

const (
	valueMask = "**masked**"
)

// GetSensored returns a copy of Config with all sensitive values masked
func (c Config) GetSensored() Config {
	c.Backend.URL = valueMask
	c.Hash.Key = valueMask
	c.Server.Profiler.Prefix = valueMask

	return c
}

func (c *Config) String() string {
	return fmt.Sprintf("%+v", c.GetSensored())
}

type Preferred map[string]string

func (p Preferred) String() string {
	var v string
	for header, value := range p {
		v += `"` + header + `" -> "` + value + `",`
	}

	if len(v) > 0 {
		v = v[0 : len(v)-1]
	}

	return v
}

func (p *Preferred) Set(v string) error {
	s := strings.SplitN(v, `=`, 2)
	if len(s) != 2 {
		return fmt.Errorf("invalid Preferred alternative argument %q, expecting <domain>=<preferred domain>", v)
	}

	if *p == nil {
		*p = make(map[string]string, 1)
	}

	if _, exists := (*p)[s[0]]; exists {
		return errors.New("duplicate preferred mapping specified")
	}

	(*p)[s[0]] = s[1]

	return nil
}

type Headers map[string]string

func (h Headers) String() string {
	var v string
	for header, value := range h {
		v += `"` + header + `:` + value + `",`
	}

	if len(v) > 0 {
		v = v[0 : len(v)-1]
	}

	return v
}

func (h *Headers) Set(v string) error {
	s := strings.SplitN(v, `:`, 2)
	if len(s) != 2 {
		return fmt.Errorf("invalid Header argument %q, expecting <header name>:<header value>", v)
	}

	if *h == nil {
		*h = make(map[string]string, 1)
	}

	(*h)[s[0]] = s[1]

	return nil
}

var (
	_ encoding.TextUnmarshaler
)

type ValidatorType string

func (vt ValidatorType) String() string {
	return string(vt)
}

func (vt *ValidatorType) Set(v string) error {
	*vt = ValidatorType(v)
	return nil
}

type ValidatorTypes []ValidatorType

func (v ValidatorTypes) AsStringSlice() []string {
	var result = make([]string, 0, len(v))
	for _, part := range v {
		result = append(result, string(part))
	}

	return result
}

func (vt *ValidatorType) UnmarshalText(value []byte) error {
	var validTypes = ValidatorTypes{VTStructure, VTLookup}

	v := string(value)
	for _, t := range validTypes.AsStringSlice() {
		if t == v {
			*vt = ValidatorType(v)
			return nil
		}
	}

	expected := strings.Join(validTypes.AsStringSlice(), ", ")
	return fmt.Errorf("unsupported value %q for validator type. Expected one of: %q", value, expected)
}

type Duration struct {
	duration time.Duration
}

func (d Duration) String() string {
	return d.duration.String()
}

func (d *Duration) Set(v string) error {
	var err error
	d.duration, err = time.ParseDuration(v)
	return err
}

func (d Duration) AsDuration() time.Duration {
	return d.duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.duration, err = time.ParseDuration(string(text))
	return err
}

type LogFormat string

func (vt LogFormat) String() string {
	return string(vt)
}

func (vt *LogFormat) Set(v string) error {
	*vt = LogFormat(v)
	return nil
}

func (vt *LogFormat) UnmarshalText(value []byte) error {
	validTypes := []string{string(LFJSON), string(LGText)}
	v := string(value)
	for _, t := range validTypes {
		if t == v {
			*vt = LogFormat(v)
			return nil
		}
	}

	expected := strings.Join(validTypes, ", ")
	return fmt.Errorf("unsupported value %q for log format. Expected one of: %q", value, expected)
}
