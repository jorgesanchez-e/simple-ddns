package route53

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/jorgesanchez-e/simple-ddns/internal/domain/dns"
)

const (
	awsAccountsPath string = "ddns.dns-server.aws"
)

type configDecoder interface {
	Decode(node string, item any) error
}

type messageLogger interface {
	Debug(msg string)
	Warning(msg string)
}

type recordsConfig struct {
	FQDN string `yaml:"fqdn"`
	Type string `yaml:"type"`
}

type zoneConfig struct {
	ID      string          `yaml:"id"`
	Records []recordsConfig `yaml:"records"`
}

type awsAccountConfig struct {
	Account         string       `yaml:"account"`
	CredentialsFile string       `yaml:"credentials-file"`
	Zones           []zoneConfig `yaml:"zones"`
}

type r53Updater interface {
	ChangeResourceRecordSets(
		ctx context.Context,
		params *route53.ChangeResourceRecordSetsInput,
		optFns ...func(*route53.Options),
	) (*route53.ChangeResourceRecordSetsOutput, error)
}

type updater struct {
	awsAccountName string
	zones          []zoneConfig
	client         r53Updater
}

func New(ctx context.Context, cnf configDecoder, logger messageLogger, accountName string) (dns.Updater, error) {
	awsAccounts := []awsAccountConfig{}
	err := cnf.Decode(awsAccountsPath, &awsAccounts)
	if err != nil {
		return nil, err
	}

	for _, account := range awsAccounts {
		if account.Account == accountName {
			awsCnf, err := awsConfig.LoadDefaultConfig(
				ctx,
				awsConfig.WithSharedConfigFiles(
					[]string{account.CredentialsFile},
				),
			)
			if err != nil {
				return nil, err
			}

			zones := []zoneConfig{}
			zones = append(zones, account.Zones...)

			return &updater{
				awsAccountName: accountName,
				zones:          zones,
				client:         route53.NewFromConfig(awsCnf),
			}, nil
		}
	}

	return nil, fmt.Errorf("account %s doesn't exist", accountName)
}

type batch struct {
	zoneID      string
	changeBatch *types.ChangeBatch
}

func (u *updater) buildBatches(records []dns.DomainRecord) []batch {
	batches := make([]batch, 0)

	for _, zone := range u.zones {
		btc := batch{
			zoneID: zone.ID,
			changeBatch: &types.ChangeBatch{
				Comment: aws.String(fmt.Sprintf("changes for zone id %s", zone.ID)),
			},
		}

		changes := make([]types.Change, 0)
		for _, zrecord := range zone.Records {
			for _, rec := range records {
				if rec.FQDN == zrecord.FQDN && rec.Type == dns.RecordType(zrecord.Type) {
					changes = append(changes, types.Change{
						Action: types.ChangeActionUpsert,
						ResourceRecordSet: &types.ResourceRecordSet{
							Name: aws.String(rec.FQDN),
							Type: types.RRType(rec.Type),
							TTL:  aws.Int64(300),
						},
					})
				}
			}
		}

		if len(changes) > 0 {
			btc.changeBatch.Changes = changes
			batches = append(batches, btc)
		}
	}

	if len(batches) > 0 {
		return batches
	}

	return nil
}

func (u *updater) UpdateDomains(ctx context.Context, records []dns.DomainRecord) error {
	errs := []error{}
	batches := u.buildBatches(records)
	for _, batch := range batches {
		payload := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  batch.changeBatch,
			HostedZoneId: &batch.zoneID,
		}

		_, err := u.client.ChangeResourceRecordSets(ctx, payload)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.New("some records couldn't be updated")
	}

	return nil
}
