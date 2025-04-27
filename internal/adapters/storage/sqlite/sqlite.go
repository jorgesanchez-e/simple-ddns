package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jorgesanchez-e/simple-ddns/internal/domain/dns"
	"github.com/jorgesanchez-e/simple-ddns/internal/domain/storage/ddns"
)

const (
	databasePath string = "ddns.storage.sqlite.db"
)

var (
	once sync.Once
)

type configDecoder interface {
	Decode(node string, item any) error
}

type messageLogger interface {
	Debug(msg string)
	Warning(msg string)
}

type sqlDriver interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
}

type store struct {
	driver sqlDriver
	logger messageLogger
}

func New(cnf configDecoder, logger messageLogger) (ddns.Controller, error) {
	dbPath := ""

	err := cnf.Decode(databasePath, &dbPath)
	if err != nil {
		return nil, err
	}

	st := store{}
	once.Do(func() {
		db := &sql.DB{}
		db, err = sql.Open("sqlite3", dbPath)
		if err == nil {
			st.driver = db
			st.logger = logger
		}
	})

	if err != nil {
		return nil, err
	}

	if err = st.createTable(); err != nil {
		return nil, err
	}

	return &st, nil
}

func (st *store) createTable() error {
	if _, err := st.driver.Exec(createTable); err != nil {
		st.driver.Close()
		return err
	}

	return nil
}

func (st *store) UpdateRecord(ctx context.Context, record dns.DomainRecord) error {
	tx, err := st.driver.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, deactivateRecord, record.FQDN, record.Type); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, insertRecord, record.FQDN, "now()", record.Type, record.Value, 1); err != nil {
		return err
	}

	tx.Commit()

	return nil
}

func (st *store) GetRecords(ctx context.Context) ([]dns.DomainRecord, error) {
	rows, err := st.driver.QueryContext(ctx, lastRecords)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []dns.DomainRecord{}
	for rows.Next() {
		record := dns.DomainRecord{}
		if err = rows.Scan(&record.FQDN, &record.Value, &record.Type); err != nil {
			st.logger.Warning(fmt.Sprintf("unable to get values err:%s", err.Error()))
			continue
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records, nil
}

func (st *store) InitRecords(ctx context.Context, records []dns.DomainRecord) error {
	tx, err := st.driver.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})

	if err != nil {
		return err
	}

	defer tx.Rollback()

	for _, record := range records {
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err = tx.ExecContext(ctx, insertRecord, record.FQDN, now, record.Value, record.Type, 1); err != nil {
			return err
		}
	}

	tx.Commit()

	return nil
}
