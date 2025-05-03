package config

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	fileOk    string = "/etc/simple-ddns/full-file.yaml"
	fileError string = "/etc/simple-ddns/file-with-error.yaml"
)

var (
	content = map[string]string{
		fileOk: `
---
ddns:
  storage:
     sqlite:
      db: /var/simple-ddns.db
  public-ip-api:
     ipify:
      check-period-mins: 1
      parse-float-32: 1.0
      parse-float-64: 2.0
      no-supported-type: "some-value"
      ipv4:
        endpoint: https://api.ipify.org
      ipv6:
        endpoint: https://api6.ipify.org
        `,
		fileError: `
    ---
      ddns:
        storage:
          sqlite:
            db: /var/simple-ddns.db      
    `,
	}
)

func TestRead(t *testing.T) {
	testCases := []struct {
		name          string
		file          string
		contentFs     afero.Fs
		expectedError error
	}{
		{
			name:          "file-ok",
			file:          fileOk,
			contentFs:     createFS(t, fileOk, content[fileOk]),
			expectedError: nil,
		},
		{
			name:          "file-error",
			file:          fileOk,
			contentFs:     createFS(t, fileOk, content[fileError]),
			expectedError: viper.ConfigParseError{},
		},
	}

	for _, tc := range testCases {
		cnf := config{}
		cnf.vp = viper.New()
		cnf.vp.SetFs(tc.contentFs)
		fileName := tc.file
		expectedError := tc.expectedError

		t.Run(tc.name, func(t *testing.T) {
			err := cnf.read(fileName)

			if err != nil {
				assert.ErrorAs(t, err, &expectedError)
			} else {
				assert.Equal(t, expectedError, err)
			}
		})

	}
}

func createFS(t *testing.T, file, content string) afero.Fs {
	t.Helper()

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(filepath.Dir(file)+"/", 0o777)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Create(file)
	if err != nil {
		t.Fatal(err)
	}

	if err = afero.WriteFile(fs, file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return fs
}

func TestDecode(t *testing.T) {
	var (
		expectedStringResult    string            = "/var/simple-ddns.db\n"
		expectedIntResult       int               = 1
		expectedFloat32Result   float32           = 1.0
		expectedFloat64Result   float64           = 2.0
		expectedMapStringString map[string]string = map[string]string{"key": "value"}
	)

	cases := []struct {
		name           string
		file           string
		nodeConfig     string
		contentFs      afero.Fs
		expectedResult any
		expectedError  error
	}{
		{
			name:           "normal-string-test",
			file:           fileOk,
			nodeConfig:     "ddns.storage.sqlite.db",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: &expectedStringResult,
			expectedError:  nil,
		},
		{
			name:           "normal-int-test",
			file:           fileOk,
			nodeConfig:     "ddns.public-ip-api.ipify.check-period-mins",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: &expectedIntResult,
			expectedError:  nil,
		},
		{
			name:           "normal-float32-test",
			file:           fileOk,
			nodeConfig:     "ddns.public-ip-api.ipify.parse-float-32",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: &expectedFloat32Result,
			expectedError:  nil,
		},
		{
			name:           "normal-float64-test",
			file:           fileOk,
			nodeConfig:     "ddns.public-ip-api.ipify.parse-float-64",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: &expectedFloat64Result,
			expectedError:  nil,
		},
		{
			name:           "node-not-found",
			file:           fileOk,
			nodeConfig:     "ddns.public-ip-api.ipify.non-existent",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: nil,
			expectedError:  fmt.Errorf("node ddns.public-ip-api.ipify.non-existent not found"),
		},
		{
			name:           "no-supported-type",
			file:           fileOk,
			nodeConfig:     "ddns.public-ip-api.ipify.no-supported-type",
			contentFs:      createFS(t, fileOk, content[fileOk]),
			expectedResult: &expectedMapStringString,
			expectedError:  fmt.Errorf("type **map[string]string no supported"),
		},
		{
			name:       "normal-struct-test",
			file:       fileOk,
			nodeConfig: "ddns.public-ip-api",
			contentFs:  createFS(t, fileOk, content[fileOk]),
			expectedResult: &publicIpConfig{
				IPify: ipify{
					CheckPeriodMins:      1,
					ParseFloat32:         1,
					ParseFloat64:         2,
					ParseNoSupportedType: "some-value",
					IPV4: ipv4{
						Endpoint: "https://api.ipify.org",
					},
					IPV6: ipv6{
						Endpoint: "https://api6.ipify.org",
					},
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range cases {
		file := tc.file
		node := tc.nodeConfig
		expectedResult := tc.expectedResult
		expectedError := tc.expectedError
		fs := tc.contentFs

		t.Run(tc.name, func(t *testing.T) {
			cnf := config{vp: viper.New()}
			cnf.vp.SetFs(fs)

			err := cnf.read(file)
			assert.NoError(t, err)

			toDecode := newTestVariable(t, expectedResult)
			err = cnf.Decode(node, toDecode)

			if err != nil {
				assert.Equal(t, expectedError, err)
			} else {
				assert.IsType(t, expectedResult, toDecode)
				assert.EqualValues(t, dereferenceValues(t, expectedResult), dereferenceValues(t, toDecode))
			}
		})
	}
}

type ipv4 struct {
	Endpoint string
}

type ipv6 struct {
	Endpoint string
}

type ipify struct {
	CheckPeriodMins      int     `yaml:"check-period-mins"`
	ParseFloat32         float32 `yaml:"parse-float-32"`
	ParseFloat64         float64 `yaml:"parse-float-64"`
	ParseNoSupportedType string  `yaml:"no-supported-type"`
	IPV4                 ipv4
	IPV6                 ipv6
}

type publicIpConfig struct {
	IPify ipify
}

func newTestVariable(t *testing.T, vType any) any {
	t.Helper()

	switch vType.(type) {
	case *string:
		newVar := ""
		return &newVar
	case *int:
		newVar := 0
		return &newVar
	case *float32:
		var newVar float32 = 0
		return &newVar
	case *float64:
		var newVar float64 = 0
		return &newVar
	case nil:
		return nil
	case *publicIpConfig:
		newStruct := publicIpConfig{}
		return &newStruct
	case *map[string]string:
		newMap := new(map[string]string)
		return &newMap
	default:
		return nil
	}
}

func dereferenceValues(t *testing.T, value any) any {
	t.Helper()

	if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr {
		switch v := v.Elem().Interface().(type) {
		case string:
			return v
		case int:
			return v
		case float32:
			return v
		case float64:
			return v
		case struct{}:
			return v
		case map[string]any:
			return v
		case any:
			return v
		default:
			return nil
		}
	}

	return nil
}

func TestParseArguments(t *testing.T) {
	var (
		fileOk string = "file-for-config"
	)

	testCases := []struct {
		name           string
		arguments      []string
		expectedResult *string
	}{
		{
			name:           "normal-case",
			arguments:      []string{"app-name", "-config", "file-for-config"},
			expectedResult: &fileOk,
		},
		// TODO: Test when no flags are defined
	}

	for _, tc := range testCases {
		args := tc.arguments
		expectedResult := tc.expectedResult
		t.Run(tc.name, func(t *testing.T) {
			originalArgs := os.Args
			defer func() {
				os.Args = originalArgs
			}()

			os.Args = args
			result := parseArguments()

			assert.Equal(t, expectedResult, result)
		})
	}
}
