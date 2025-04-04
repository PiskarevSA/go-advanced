package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type PgStorage struct {
	MemStorage
	databaseDSN string
	conn        *pgx.Conn
}

func NewPgStorage(databaseDSN string) *PgStorage {
	return &PgStorage{
		MemStorage:  *NewMemStorage(),
		databaseDSN: databaseDSN,
	}
}

func (p *PgStorage) Ping() error {
	if err := p.ensureConnected(); err != nil {
		return err
	}
	return p.conn.Ping(context.Background())
}

func (p *PgStorage) Close() error {
	if p.conn != nil {
		return p.conn.Close(context.Background())
	}
	return nil
}

func (p *PgStorage) ensureConnected() error {
	if p.conn == nil {
		var err error
		p.conn, err = pgx.Connect(context.Background(), p.databaseDSN)
		return err
	}
	return nil
}
