package ipify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	publicip "github.com/jorgesanchez-e/simple-ddns/internal/domain/public-ip"
	"github.com/stretchr/testify/assert"
)

const (
	successfulIPV4URLTest      string = "https://ipify/v4/successful-case"
	err404IPV4URLTest          string = "https://ipify/v4/404"
	errHttpErrIPV4Request      string = "https://ipify/v4/http-error"
	errHttpIPV4BodyFormatError string = "https://ipify/v4/http-body-error"

	successfulIPV6URLTest      string = "https://ipify/v6/successful-case"
	err404IPV6URLTest          string = "https://ipify/v6/404"
	errHttpErrIPV6Request      string = "https://ipify/v6/http-error"
	errHttpIPV6BodyFormatError string = "https://ipify/v6/http-body-error"
)

type httpRequestorMock struct {
	t *testing.T
}

func (httpMock httpRequestorMock) Do(req *http.Request) (*http.Response, error) {
	httpMock.t.Helper()

	switch req.URL.String() {
	case successfulIPV4URLTest:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("127.0.0.1"))),
		}, nil
	case successfulIPV6URLTest:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("::1"))),
		}, nil
	case err404IPV4URLTest:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}, nil
	case err404IPV6URLTest:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}, nil
	case errHttpErrIPV4Request:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}, fmt.Errorf("http request error for ipv4")
	case errHttpErrIPV6Request:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}, fmt.Errorf("http request error for ipv6")
	case errHttpIPV4BodyFormatError:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{"))),
		}, nil
	case errHttpIPV6BodyFormatError:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{"))),
		}, nil
	default:
		httpMock.t.Fatal("not implemented test")
	}

	return nil, nil
}

type messageLoggerMock struct {
	errorMessages []string
	infoMessages  []string
	debugMessages []string
	t             *testing.T
}

func (loggerMock *messageLoggerMock) Error(err error) {
	loggerMock.t.Helper()
	loggerMock.errorMessages = append(loggerMock.errorMessages, err.Error())
}

func (loggerMock *messageLoggerMock) Info(msg string) {
	loggerMock.t.Helper()
	loggerMock.infoMessages = append(loggerMock.infoMessages, msg)
}

func (loggerMock *messageLoggerMock) Debug(msg string) {
	loggerMock.t.Helper()
	loggerMock.debugMessages = append(loggerMock.debugMessages, msg)
}

func TestGetIP(t *testing.T) {
	testCases := []struct {
		name                  string
		getter                ipifyGetter
		expectedErrMessages   []string
		expectedInfoMessages  []string
		expectedDebugMessages []string
		expectedResult        publicip.IP
	}{
		{
			name: "successful case",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: successfulIPV4URLTest},
					IPV6:              endpoint{URL: successfulIPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: stringPointer("127.0.0.1"),
				V6: stringPointer("::1"),
			},
			expectedErrMessages:   []string{},
			expectedInfoMessages:  []string{},
			expectedDebugMessages: []string{"ipify: ipv4: 127.0.0.1", "ipify: ipv6: ::1"},
		},
		{
			name: "ipv4 not found, ipv6 ok",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: err404IPV4URLTest},
					IPV6:              endpoint{URL: successfulIPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: nil,
				V6: stringPointer("::1"),
			},
			expectedErrMessages: []string{
				"ipify: url=https://ipify/v4/404 http error: httpd code 404",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv6: ::1",
			},
		},
		{
			name: "ipv6 not found, ipv4 ok",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: successfulIPV4URLTest},
					IPV6:              endpoint{URL: err404IPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: stringPointer("127.0.0.1"),
				V6: nil,
			},
			expectedErrMessages: []string{
				"ipify: url=https://ipify/v6/404 http error: httpd code 404",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv4: 127.0.0.1",
			},
		},
		{
			name: "ipv6 not found, either ipv4",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: err404IPV4URLTest},
					IPV6:              endpoint{URL: err404IPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: nil,
				V6: nil,
			},
			expectedErrMessages: []string{
				"ipify: url=https://ipify/v4/404 http error: httpd code 404",
				"ipify: url=https://ipify/v6/404 http error: httpd code 404",
			},
			expectedInfoMessages:  []string{},
			expectedDebugMessages: []string{},
		},
		{
			name: "ipv4 http error",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: errHttpErrIPV4Request},
					IPV6:              endpoint{URL: successfulIPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: nil,
				V6: stringPointer("::1"),
			},
			expectedErrMessages: []string{
				"ipify: url=https://ipify/v4/http-error network error: http request error for ipv4",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv6: ::1",
			},
		},
		{
			name: "ipv6 http error",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: successfulIPV4URLTest},
					IPV6:              endpoint{URL: errHttpErrIPV6Request},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: stringPointer("127.0.0.1"),
				V6: nil,
			},
			expectedErrMessages: []string{
				"ipify: url=https://ipify/v6/http-error network error: http request error for ipv6",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv4: 127.0.0.1",
			},
		},
		{
			name: "ipv4 body format error",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: errHttpIPV4BodyFormatError},
					IPV6:              endpoint{URL: successfulIPV6URLTest},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: nil,
				V6: stringPointer("::1"),
			},
			expectedErrMessages: []string{
				"invalid ipv4 format {, err:Key: '' Error:Field validation for '' failed on the 'ipv4' tag",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv6: ::1",
			},
		},
		{
			name: "ipv6 body format error",
			getter: ipifyGetter{
				config: ipifyConfig{
					CheckPeriodInMins: 1,
					IPV4:              endpoint{URL: successfulIPV4URLTest},
					IPV6:              endpoint{URL: errHttpIPV6BodyFormatError},
				},
				client: httpRequestorMock{t: t},
			},
			expectedResult: publicip.IP{
				V4: stringPointer("127.0.0.1"),
				V6: nil,
			},
			expectedErrMessages: []string{
				"invalid ipv6 format {, err:Key: '' Error:Field validation for '' failed on the 'ipv6' tag",
			},
			expectedInfoMessages: []string{},
			expectedDebugMessages: []string{
				"ipify: ipv4: 127.0.0.1",
			},
		},
	}

	for _, tc := range testCases {
		ipfyGetter := tc.getter
		expectedResult := tc.expectedResult
		expectedErrorMessages := tc.expectedErrMessages
		expectedInfoMessages := tc.expectedInfoMessages
		expectedDebugMessages := tc.expectedDebugMessages

		logger := &messageLoggerMock{
			t:             t,
			errorMessages: make([]string, 0),
			infoMessages:  make([]string, 0),
			debugMessages: make([]string, 0),
		}
		ipfyGetter.logger = logger

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			result := ipfyGetter.GetIP(ctx)

			assert.Equal(t, expectedResult, result)
			assert.Equal(t, expectedErrorMessages, logger.errorMessages)
			assert.Equal(t, expectedInfoMessages, logger.infoMessages)
			assert.Equal(t, expectedDebugMessages, logger.debugMessages)
		})
	}
}

func stringPointer(str string) *string {
	return &str
}
