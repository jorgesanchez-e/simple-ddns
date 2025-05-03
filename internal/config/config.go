package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

const (
	configFileType    string = "yaml"
	errReadConfigFile string = "unable to read config file"
)

type config struct {
	vp *viper.Viper
}

func New() (*config, error) {
	var configFile *string
	if configFile = parseArguments(); configFile == nil {
		return nil, errors.New(errReadConfigFile)
	}

	cnf := new(config)
	cnf.vp = viper.New()
	if err := cnf.read(*configFile); err != nil {
		return nil, err
	}

	return cnf, nil
}

func parseArguments() *string {
	configFile := flag.String("config", "", "-config=<CONFIG-FILE-PATH>")
	flag.Parse()

	if *configFile == "" {
		return nil
	}

	return configFile
}

func (c *config) read(file string) error {
	c.vp.SetConfigType(configFileType)
	c.vp.SetConfigFile(file)
	if err := c.vp.ReadInConfig(); err != nil {
		return fmt.Errorf("%s: %w", errReadConfigFile, err)
	}

	return nil
}

func (c *config) Decode(node string, item any) error {
	bytes, err := c.find(node)
	if err != nil {
		return err
	}

	if len(bytes) == 0 {
		return fmt.Errorf("node %s not found", node)
	}

	if v := reflect.ValueOf(item); v.Kind() == reflect.Ptr {
		switch v = v.Elem(); v.Kind() {
		case reflect.Slice, reflect.Struct, reflect.Map:
			if err = yaml.Unmarshal(bytes, item); err != nil {
				return err
			}
		case reflect.String:
			v.SetString(string(bytes))
		case reflect.Int:
			var value int = 0
			if value, err = strconv.Atoi(strings.TrimSpace(string(bytes))); err != nil {
				return fmt.Errorf("unable to decode int value, err:%w", err)
			}
			v.SetInt(int64(value))
		case reflect.Float32:
			var n float64 = 0
			if n, err = strconv.ParseFloat(strings.TrimSpace(string(bytes)), 32); err != nil {
				return fmt.Errorf("unable to decode float32 value, err:%w", err)
			}
			v.SetFloat(n)
		case reflect.Float64:
			var n float64 = 0
			if n, err = strconv.ParseFloat(strings.TrimSpace(string(bytes)), 64); err != nil {
				return fmt.Errorf("unable to decode float64 value, err:%w", err)
			}
			v.SetFloat(n)
		default:
			return fmt.Errorf("type %T no supported", item)
		}
	} else {
		return fmt.Errorf("item is not a pointer")
	}

	return nil
}

func (c *config) find(node string) ([]byte, error) {
	if c == nil {
		return nil, errors.New("config haven't been read")
	}

	if c.vp.Get(node) == nil {
		return nil, fmt.Errorf("node %s not found", node)
	}

	buf := new(bytes.Buffer)
	if err := yaml.NewEncoder(buf).Encode(c.vp.Get(node)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
