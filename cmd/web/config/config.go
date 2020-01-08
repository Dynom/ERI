package config

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
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
			Resolver string `toml:"resolver"`
		} `toml:"validator"`
		Profiler struct {
			Enable bool   `toml:"enable"`
			Prefix string `toml:"prefix"`
		} `toml:"profiler"`
		Backend struct {
			Driver string `toml:"driver"`
			URL    string `toml:"url"`
		} `toml:"backend"`
	} `toml:"server"`
}

type Header struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}
