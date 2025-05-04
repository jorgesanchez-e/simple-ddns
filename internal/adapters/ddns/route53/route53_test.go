package route53

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/jorgesanchez-e/simple-ddns/internal/domain/dns"
	"github.com/stretchr/testify/assert"
)

func Test_buildBatches(t *testing.T) {
	testCases := []struct {
		name            string
		updater         updater
		records         []dns.DomainRecord
		expectedBatches []batch
	}{
		{
			name: "build-batches-ok",
			updater: updater{
				zones: []zoneConfig{
					{
						ID: "1111111111111111111111",
						Records: []recordsConfig{
							{
								FQDN: "www.local-environment.com",
								Type: "A",
							},
							{
								FQDN: "jenkins.local-environment.com",
								Type: "A",
							},
							{
								FQDN: "www6.local-environment.com",
								Type: "AAAA",
							},
							{
								FQDN: "jenkins6.local-environment.com",
								Type: "AAAA",
							},
						},
					},
				},
			},
			records: []dns.DomainRecord{
				{
					FQDN: "jenkins.local-environment.com",
					Type: dns.A,
				},
				{
					FQDN: "www6.local-environment.com",
					Type: "AAAA",
				},
			},
			expectedBatches: []batch{
				{
					zoneID: "1111111111111111111111",
					changeBatch: &types.ChangeBatch{
						Comment: aws.String("changes for zone id 1111111111111111111111"),
						Changes: []types.Change{
							{
								Action: types.ChangeActionUpsert,
								ResourceRecordSet: &types.ResourceRecordSet{
									Name: aws.String("jenkins.local-environment.com"),
									Type: types.RRTypeA,
									TTL:  aws.Int64(300),
								},
							},
							{
								Action: types.ChangeActionUpsert,
								ResourceRecordSet: &types.ResourceRecordSet{
									Name: aws.String("www6.local-environment.com"),
									Type: types.RRTypeAaaa,
									TTL:  aws.Int64(300),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no-batches",
			updater: updater{
				zones: []zoneConfig{
					{
						ID: "1111111111111111111111",
						Records: []recordsConfig{
							{
								FQDN: "www.local-environment.com",
								Type: "A",
							},
							{
								FQDN: "jenkins.local-environment.com",
								Type: "A",
							},
							{
								FQDN: "www6.local-environment.com",
								Type: "AAAA",
							},
							{
								FQDN: "jenkins6.local-environment.com",
								Type: "AAAA",
							},
						},
					},
				},
			},
			records: []dns.DomainRecord{
				{
					FQDN: "vpn.local-environment.com",
					Type: "A",
				},
			},
			expectedBatches: nil,
		},
	}

	for _, tc := range testCases {
		updater := tc.updater
		records := tc.records
		expectedBatches := tc.expectedBatches

		t.Run(tc.name, func(t *testing.T) {
			batches := updater.buildBatches(records)

			assert.Equal(t, expectedBatches, batches)
		})
	}
}

type r53MockClient struct {
	err error
}

func (mock r53MockClient) ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	return nil, mock.err
}

func Test_UpdateDomains(t *testing.T) {
	testCases := []struct {
		name          string
		updater       updater
		records       []dns.DomainRecord
		expectedError error
	}{
		{
			name: "records-updated-ok",
			updater: updater{
				awsAccountName: "main",
				zones: []zoneConfig{
					{
						ID: "000000000000000",
						Records: []recordsConfig{
							{
								FQDN: "home.google.com",
								Type: "A",
							},
						},
					},
				},
				client: r53MockClient{},
			},
			records: []dns.DomainRecord{
				{
					Type:  "A",
					Value: "192.168.100.1",
					FQDN:  "home.google.com",
				},
			},
		},
		{
			name: "update-error",
			updater: updater{
				awsAccountName: "main",
				zones: []zoneConfig{
					{
						ID: "000000000000000",
						Records: []recordsConfig{
							{
								FQDN: "home.google.com",
								Type: "A",
							},
						},
					},
				},
				client: r53MockClient{
					err: errors.New("update error"),
				},
			},
			records: []dns.DomainRecord{
				{
					Type:  "A",
					Value: "192.168.100.1",
					FQDN:  "home.google.com",
				},
			},
			expectedError: errors.New("some records couldn't be updated"),
		},
	}

	for _, tc := range testCases {
		updater := tc.updater
		expectedError := tc.expectedError
		records := tc.records

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := updater.UpdateDomains(ctx, records)

			assert.Equal(t, expectedError, err)
		})
	}
}
