package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jorgesanchez-e/simple-ddns/internal/domain/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockLogger struct {
	mock.Mock
	debugMessages   []string
	warningMessages []string
}

func (ml *mockLogger) Debug(msg string) {
	ml.Called()
	ml.debugMessages = append(ml.debugMessages, msg)
}

func (ml *mockLogger) Warning(msg string) {
	ml.Called()
	ml.warningMessages = append(ml.warningMessages, msg)
}

func TestCreateTable(t *testing.T) {
	testCases := []struct {
		name          string
		createMock    func(*testing.T) (*sql.DB, sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "create_table_without_error",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectExec(regexp.QuoteMeta(createTable)).
					WillReturnResult(sqlmock.NewResult(0, 0))

				return db, mock
			},
			expectedError: nil,
		},
		{
			name: "create_table_with_error",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectExec(regexp.QuoteMeta(createTable)).
					WillReturnError(errors.New("table permissions error"))

				return db, mock
			},
			expectedError: errors.New("table permissions error"),
		},
	}

	for _, tc := range testCases {
		db, dbMock := tc.createMock(t)
		st := store{
			driver: db,
			logger: &mockLogger{},
		}
		expectedError := tc.expectedError

		t.Run(tc.name, func(t *testing.T) {
			err := st.createTable()

			assert.Equal(t, expectedError, err)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
		db.Close()
	}
}

func TestUpdateRecord(t *testing.T) {
	testCases := []struct {
		name           string
		createMock     func(*testing.T) (*sql.DB, sqlmock.Sqlmock)
		recordToUpdate dns.DomainRecord
		expectedError  error
	}{
		{
			name: "test-record-updated-ok",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(deactivateRecord)).
					WithArgs("wwww.google.com", "A").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta(insertRecord)).
					WithArgs(
						"wwww.google.com",
						"now()",
						"A",
						"192.168.1.10",
						1,
					).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()

				return db, mock
			},
			recordToUpdate: dns.DomainRecord{
				FQDN:  "wwww.google.com",
				Type:  dns.A,
				Value: "192.168.1.10",
			},
		},
		{
			name: "test-record-updated-deactivate-fail",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(deactivateRecord)).
					WithArgs("wwww.google.com", "A").WillReturnError(errors.New("unable to deactivate record"))
				mock.ExpectRollback()

				return db, mock
			},
			recordToUpdate: dns.DomainRecord{
				FQDN:  "wwww.google.com",
				Type:  dns.A,
				Value: "192.168.1.10",
			},
			expectedError: errors.New("unable to deactivate record"),
		},
		{
			name: "test-record-updated-insert-fail",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(deactivateRecord)).
					WithArgs("wwww.google.com", "A").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta(insertRecord)).
					WithArgs(
						"wwww.google.com",
						"now()",
						"A",
						"192.168.1.10",
						1,
					).WillReturnError(errors.New("unable to insert record"))
				mock.ExpectRollback()

				return db, mock
			},
			recordToUpdate: dns.DomainRecord{
				FQDN:  "wwww.google.com",
				Type:  dns.A,
				Value: "192.168.1.10",
			},
			expectedError: errors.New("unable to insert record"),
		},
		{
			name: "test-record-updated-transaction-fail",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin().WillReturnError(errors.New("unable to create new transaction"))

				return db, mock
			},
			recordToUpdate: dns.DomainRecord{
				FQDN:  "wwww.google.com",
				Type:  dns.A,
				Value: "192.168.1.10",
			},
			expectedError: errors.New("unable to create new transaction"),
		},
	}

	for _, tc := range testCases {
		db, dbMock := tc.createMock(t)
		st := store{
			driver: db,
			logger: &mockLogger{},
		}
		record := tc.recordToUpdate
		expectedError := tc.expectedError
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := st.UpdateRecord(ctx, record)

			assert.Equal(t, expectedError, err)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
		db.Close()
	}
}

func TestGetRecords(t *testing.T) {
	testCases := []struct {
		name            string
		createMock      func(*testing.T) (*sql.DB, sqlmock.Sqlmock)
		expectedRecords []dns.DomainRecord
		expectedError   error
	}{
		{
			name: "test-get-records-query-error",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectQuery(regexp.QuoteMeta(lastRecords)).WillReturnError(errors.New("query error"))

				return db, mock
			},
			expectedRecords: nil,
			expectedError:   errors.New("query error"),
		},
		{
			name: "test-no-records",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectQuery(regexp.QuoteMeta(lastRecords)).
					WillReturnRows(
						sqlmock.NewRows([]string{"fqdn", "ip", "register_type"}),
					)

				return db, mock
			},
			expectedRecords: nil,
			expectedError:   nil,
		},
		{
			name: "test-get-records-ok",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectQuery(regexp.QuoteMeta(lastRecords)).
					WillReturnRows(
						sqlmock.NewRows([]string{"fqdn", "ip", "register_type"}).AddRow(
							"www.google.com",
							"192.168.100.1",
							"A",
						).AddRow(
							"www6.google.com",
							"2001:0db8:0000:0000:0000:8a2e:0370:7334",
							"AAAA",
						),
					)

				return db, mock
			},
			expectedRecords: []dns.DomainRecord{
				{
					Type:  dns.A,
					FQDN:  "www.google.com",
					Value: "192.168.100.1",
				},
				{
					Type:  dns.AAAA,
					FQDN:  "www6.google.com",
					Value: "2001:0db8:0000:0000:0000:8a2e:0370:7334",
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		db, dbMock := tc.createMock(t)
		logger := &mockLogger{}
		st := store{
			driver: db,
			logger: logger,
		}
		expectedError := tc.expectedError
		expectedRecords := tc.expectedRecords

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			records, err := st.GetRecords(ctx)

			assert.Equal(t, expectedError, err)
			assert.Equal(t, expectedRecords, records)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
		db.Close()
	}
}

type AnyISODate struct{}

func (a AnyISODate) Match(v driver.Value) bool {
	value, ok := v.(string)
	if !ok {
		return false
	}

	pattern := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`
	return regexp.MustCompile(pattern).Match([]byte(value))
}

func TestInitRecords(t *testing.T) {
	testCases := []struct {
		name          string
		createMock    func(*testing.T) (*sql.DB, sqlmock.Sqlmock)
		records       []dns.DomainRecord
		expectedError error
	}{
		{
			name: "create-transaction-error",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin().WillReturnError(errors.New("unable to get new transaction"))
				return db, mock
			},
			expectedError: errors.New("unable to get new transaction"),
		},
		{
			name: "init-records-insert-fail",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(insertRecord)).WithArgs("www.google.com", AnyISODate{}, "192.168.1.1", "A", 1).WillReturnError(errors.New("insert error"))
				mock.ExpectRollback()

				return db, mock
			},
			records: []dns.DomainRecord{
				{
					FQDN:  "www.google.com",
					Value: "192.168.1.1",
					Type:  "A",
				},
			},
			expectedError: errors.New("insert error"),
		},
		{
			name: "init-records-ok",
			createMock: func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(insertRecord)).WithArgs("www.google.com", AnyISODate{}, "192.168.1.1", "A", 1).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(insertRecord)).WithArgs("www6.google.com", AnyISODate{}, "2001:0db8:0000:0000:0000:8a2e:0370:7334", "AAAA", 1).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				return db, mock
			},
			records: []dns.DomainRecord{
				{
					FQDN:  "www.google.com",
					Value: "192.168.1.1",
					Type:  "A",
				},
				{
					FQDN:  "www6.google.com",
					Value: "2001:0db8:0000:0000:0000:8a2e:0370:7334",
					Type:  "AAAA",
				},
			},
		},
	}

	for _, tc := range testCases {
		db, dbMock := tc.createMock(t)
		st := store{
			driver: db,
			logger: &mockLogger{},
		}
		records := tc.records
		expectedError := tc.expectedError

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := st.InitRecords(ctx, records)

			assert.Equal(t, expectedError, err)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
		db.Close()
	}
}
