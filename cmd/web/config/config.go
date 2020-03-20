package config

import (
	"encoding"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	VTStructure ValidatorType = "structure"
	VTLookup    ValidatorType = "lookup"
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

// @todo restructure, not all elements belong beneath "Server" and separating "CORS" makes little sense as well
// Config holds central config parameters
type Config struct {
	Client struct {
		InputLengthMax uint64 `toml:"inputLengthMax"`
	} `toml:"client"`
	CORS struct {
		AllowedOrigins []string `toml:"allowedOrigins"`
	} `toml:"CORS"`
	Server struct {
		ListenOn string   `toml:"listenOn"`
		Headers  []Header `toml:"headers"`
		Log      struct {
			Level string `toml:"level"`
		} `toml:"log"`
		Hash struct {
			Key string `toml:"key"`
			//Enable bool   `toml:"enable"`
		} `toml:"hash"`
		Finder struct {
			UseBuckets bool `toml:"useBuckets"`
		} `toml:"finder"`
		Validator struct {
			Resolver         string        `toml:"resolver"`
			SuggestValidator ValidatorType `toml:"suggest"`
		} `toml:"validator"`
		Profiler struct {
			Enable bool   `toml:"enable"`
			Prefix string `toml:"prefix"`
		} `toml:"profiler"`
		//Backend struct {
		//	Driver string `toml:"driver"`
		//	URL    string `toml:"url"`
		//} `toml:"backend"`
	} `toml:"server"`
}

type Header struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

var (
	_ encoding.TextUnmarshaler
)

type ValidatorType string
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
