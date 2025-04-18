package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/spf13/viper"
)

const (
	defaultConfigFile string = "/etc/simpleddns/config.yaml"
	configFileType    string = "yaml"

	errReadConfigFile string = "unable to read config file"
)

type config struct {
	vp *viper.Viper
}

func New() (*config, error) {
	var cnf *config
	var err error
	var configFile *string

	if configFile = parseArguments(); configFile == nil {
		return nil, fmt.Errorf("%s %s", errReadConfigFile, defaultConfigFile)
	}

	cnf = new(config)
	err = cnf.read(*configFile)

	return cnf, err
}

func parseArguments() *string {
	configFile := flag.String("config", defaultConfigFile, "-config=<CONFIG-FILE-PATH>")
	flag.Parse()

	return configFile
}

func (c *config) read(file string) error {
	c.vp = viper.New()

	c.vp.SetConfigType(configFileType)
	c.vp.SetConfigFile(file)
	if err := c.vp.ReadInConfig(); err != nil {
		return fmt.Errorf("%s: %w", errReadConfigFile, err)
	}

	return nil
}

func (c *config) Find(node string) (io.Reader, error) {
	if c == nil {
		return nil, errors.New("config haven't been read")
	}

	n := c.vp.Get(node)
	if d, is := n.(map[string]interface{}); is {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(d); err != nil {
			return nil, err
		}

		return buf, nil
	} else {
		return nil, fmt.Errorf("unable to find config:%s", node)
	}
}
