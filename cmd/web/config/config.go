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
	References map[string][]string `toml:"references"`
	Client     struct {
		InputLengthMax int `toml:"inputLengthMax"`
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
		Profiler struct {
			Enable bool   `toml:"enable"`
			Prefix string `toml:"prefix"`
		} `toml:"profiler"`
	} `toml:"server"`
}

type Header struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}
