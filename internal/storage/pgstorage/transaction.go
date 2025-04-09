package pgstorage

import (
	"context"
	"errors"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func shouldRetry(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)
}

func backoff(retries int) time.Duration {
	return time.Duration(1+2*retries) * time.Second
}

func doTransaction(
	ctx context.Context, pool *pgxpool.Pool, doQueries func(pgx.Tx) error,
) error {
	select {
	case <-ctx.Done():
		return entities.NewInternalError("avoid transaction", ctx.Err())
	default:
		tx, err := pool.Begin(ctx)
		if err != nil {
			return entities.NewInternalError("failed to begin transaction", err)
		}

		defer func() { _ = tx.Rollback(ctx) }()

		if err := doQueries(tx); err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return entities.NewInternalError("failed to commit transaction", err)
		}

		return err
	}
}

func doTransactionWithRetries(
	ctx context.Context, pool *pgxpool.Pool, doQueries func(pgx.Tx) error,
) error {
	err := doTransaction(ctx, pool, doQueries)

	retries := 0
	for shouldRetry(err) && retries < retryCount {
		time.Sleep(backoff(retries))
		err = doTransaction(ctx, pool, doQueries)
		retries++
	}
	return err
}
