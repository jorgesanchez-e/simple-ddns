package ipify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"

	publicip "github.com/jorgesanchez-e/simple-ddns/internal/domain/public-ip"
)

const (
	configNode string = "ddns.storage.public-ip-api.ipify"
	ipifyIPV4  int    = iota
	ipifyIPV6
)

var ErrInvalidIpType = errors.New("invalid ip type argument")

type endpoint struct {
	URL string `yaml:"endpoint"`
}

type ipifyConfig struct {
	CheckPeriodInMins int      `yaml:"check-period-mins"`
	IPV4              endpoint `yaml:"ipv4"`
	IPV6              endpoint `yaml:"ipv4"`
}

type configDecoder interface {
	Decode(node string, item any) error
}

type httpRequestor interface {
	Do(req *http.Request) (*http.Response, error)
}

type messageLogger interface {
	Error(err error)
	Info(msg string)
	Debug(msg string)
}

type ipifyGetter struct {
	config ipifyConfig
	client httpRequestor
	logger messageLogger
}

func New(config configDecoder, logger messageLogger) (*ipifyGetter, error) {
	cnf := ipifyConfig{}
	err := config.Decode(configNode, &cnf)
	if err != nil {
		return nil, fmt.Errorf("ipify: unabel to create new ipify instance, err:%w", err)
	}

	return &ipifyGetter{
		config: cnf,
		client: http.DefaultClient,
		logger: logger,
	}, nil
}

func (ipi *ipifyGetter) GetIP(ctx context.Context) (publicIp publicip.IP) {
	var err error
	defer func() {
		if publicIp.V4 != nil {
			ipi.logger.Debug(fmt.Sprintf("ipify: ipv4: %s", *publicIp.V4))
		}
		if publicIp.V6 != nil {
			ipi.logger.Debug(fmt.Sprintf("ipify: ipv6: %s", *publicIp.V6))
		}
	}()

	if publicIp.V4, err = ipi.getIP(ctx, ipifyIPV4); err != nil {
		ipi.logger.Error(err)
	}

	if publicIp.V6, err = ipi.getIP(ctx, ipifyIPV6); err != nil {
		ipi.logger.Error(err)
	}

	return publicIp
}

func (ipi *ipifyGetter) getIP(ctx context.Context, ipType int) (_ *string, err error) {
	url := ""
	switch ipType {
	case ipifyIPV4:
		url = ipi.config.IPV4.URL
	case ipifyIPV6:
		url = ipi.config.IPV6.URL
	default:
		return nil, ErrInvalidIpType
	}

	req := &http.Request{}
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, fmt.Errorf("ipify: url=%s request build error: %w", url, err)
	}

	req = req.WithContext(ctx)
	res := &http.Response{}

	if res, err = ipi.client.Do(req); err != nil {
		return nil, fmt.Errorf("ipify: url=%s network error: %w", url, err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipify: url=%s http error: %w", url, fmt.Errorf("httpd code %d", res.StatusCode))
	}

	ip, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ipify: url=%s response body error: %w", url, err)
	}

	ipStr := string(ip)
	if ipType == ipifyIPV4 {
		if err = validator.New().Var(ipStr, "ipv4"); err != nil {
			return nil, fmt.Errorf("invalid ipv4 format %s, err:%w", ipStr, err)
		}
		return &ipStr, nil
	}

	if err = validator.New().Var(ipStr, "ipv6"); err != nil {
		return nil, fmt.Errorf("invalid ipv6 format %s, err:%w", ipStr, err)
	}

	return &ipStr, nil
}
