package dns

import "context"

const (
	A    RecordType = "A"
	AAAA RecordType = "AAAA"
)

type RecordType string

type DomainRecord struct {
	Type  RecordType
	Value string
	FQDN  string
}

type Updater interface {
	UpdateDomains(context.Context, []DomainRecord) error
}
