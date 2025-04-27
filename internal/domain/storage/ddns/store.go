package ddns

import (
	"context"

	"github.com/jorgesanchez-e/simple-ddns/internal/domain/dns"
)

type Controller interface {
	UpdateRecord(context.Context, dns.DomainRecord) error
	GetRecords(context.Context) ([]dns.DomainRecord, error)
	InitRecords(context.Context, []dns.DomainRecord) error
}
