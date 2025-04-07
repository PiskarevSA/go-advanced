package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const retryCount = 3

type PgStorage struct {
	databaseDSN string
	pool        *pgxpool.Pool
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func NewPgStorage(ctx context.Context, databaseDSN string) (*PgStorage, error) {
	result := &PgStorage{
		databaseDSN: databaseDSN,
	}
	if err := result.connect(ctx); err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	if err := result.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return result, nil
}

func shouldRetry(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)
}

func backoff(retries int) time.Duration {
	return time.Duration(1+2*retries) * time.Second
}

func (s *PgStorage) queryRowWithRetries(
	ctx context.Context, optionalTx pgx.Tx, query string, valuePtr any, args ...any,
) error {
	var err error

	doQuery := func() {
		var row pgx.Row
		if optionalTx != nil {
			row = optionalTx.QueryRow(ctx, query, args...)
		} else {
			row = s.pool.QueryRow(ctx, query, args...)
		}
		err = row.Scan(valuePtr)
	}

	doQuery()

	retries := 0
	for shouldRetry(err) && retries < retryCount {
		time.Sleep(backoff(retries))
		doQuery()
		retries++
	}
	return err
}

func (s *PgStorage) queryRowsWithRetries(
	ctx context.Context, tx pgx.Tx, query string,
	makeScanDest func() []any, saveScanDest func(), args ...any,
) error {
	var err error

	doQuery := func() {
		var rows pgx.Rows
		rows, err = tx.Query(ctx, query, args...)
		if err != nil {
			err = entities.NewInternalError("sql query error: "+err.Error(), err)
		}
		defer rows.Close()

		for rows.Next() {
			dest := makeScanDest()
			if err = rows.Scan(dest...); err != nil {
				err = entities.NewInternalError("sql query error: "+err.Error(), err)
			}
			saveScanDest()
		}
		if rows.Err() != nil {
			err = entities.NewInternalError("sql query error: "+rows.Err().Error(), err)
		}
	}

	doQuery()

	retries := 0
	for shouldRetry(err) && retries < retryCount {
		time.Sleep(backoff(retries))
		doQuery()
		retries++
	}
	return err
}

func (s *PgStorage) GetMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	switch metric.Type {
	case entities.MetricTypeGauge:
		query := "select value from gauge where name = $1"
		var value entities.Gauge
		err := s.queryRowWithRetries(
			ctx, nil, query, &value, metric.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		} else if err != nil {
			return nil, entities.NewInternalError("sql query error: "+err.Error(), err)
		}
		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		query := "select value from counter where name = $1"
		var value entities.Counter
		err := s.queryRowWithRetries(
			ctx, nil, query, &value, metric.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		} else if err != nil {
			return nil, entities.NewInternalError("sql query error: "+err.Error(), err)
		}
		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: 0,
			Delta: value,
		}
		return &result, nil
	}
	return nil, entities.NewInternalError(
		"unexpected internal metric type: "+metric.Type.String(), nil)
}

func (s *PgStorage) UpdateMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	return s.updateMetric(ctx, nil, metric)
}

func (s *PgStorage) updateMetric(
	ctx context.Context, optionalTx pgx.Tx, metric entities.Metric,
) (*entities.Metric, error) {
	switch metric.Type {
	case entities.MetricTypeGauge:
		query := joinLines(
			"insert into gauge (name, value)",
			"values ($1, $2)",
			"on conflict(name)",
			"do update set",
			"  value = excluded.value",
			"returning value")
		var value entities.Gauge
		err := s.queryRowWithRetries(
			ctx, optionalTx, query, &value, metric.Name, metric.Value)
		if err != nil {
			return nil, entities.NewInternalError("sql query error: "+err.Error(), err)
		}

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		query := joinLines(
			"insert into counter (name, value)",
			"values ($1, $2)",
			"on conflict(name)",
			"do update set",
			"  value = counter.value + excluded.value",
			"returning value")
		var value entities.Counter
		err := s.queryRowWithRetries(
			ctx, optionalTx, query, &value, metric.Name, metric.Delta)
		if err != nil {
			return nil, entities.NewInternalError("sql query error: "+err.Error(), err)
		}

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: 0,
			Delta: value,
		}
		return &result, nil
	}
	return nil, entities.NewInternalError(
		"unexpected internal metric type: "+metric.Type.String(), nil)
}

func (s *PgStorage) UpdateMetrics(ctx context.Context, metrics []entities.Metric,
) ([]entities.Metric, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, entities.NewInternalError("database error: "+err.Error(), err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := make([]entities.Metric, 0)
	for i, metric := range metrics {
		updatedMetric, err := s.updateMetric(ctx, tx, metric)
		if err != nil {
			return nil, fmt.Errorf("metric[%v]: %w", i, err)
		}
		result = append(result, *updatedMetric)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, entities.NewInternalError("database error: "+err.Error(), err)
	}
	return result, nil
}

func (s *PgStorage) GetMetricsByTypes(ctx context.Context,
	gauge map[entities.MetricName]entities.Gauge,
	counter map[entities.MetricName]entities.Counter,
) error {
	// open transaction to ensure time consistency between gauges and counters
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entities.NewInternalError("database error: "+err.Error(), err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := "select name, value from gauge"
	var name entities.MetricName
	var gaugeValue entities.Gauge
	makeScanDest := func() []any { return []any{&name, &gaugeValue} }
	saveScanDest := func() { gauge[name] = gaugeValue }
	err = s.queryRowsWithRetries(ctx, tx, query, makeScanDest, saveScanDest)
	if err != nil {
		return err
	}

	query = "select name, value from counter"
	var counterValue entities.Counter
	makeScanDest = func() []any { return []any{&name, &counterValue} }
	saveScanDest = func() { counter[name] = counterValue }
	err = s.queryRowsWithRetries(ctx, tx, query, makeScanDest, saveScanDest)
	if err != nil {
		return err
	}

	return nil
}

func (s *PgStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *PgStorage) Close(ctx context.Context) error {
	s.pool.Close()
	return nil
}

func (s *PgStorage) connect(ctx context.Context) error {
	var err error
	s.pool, err = pgxpool.New(ctx, s.databaseDSN)
	return err
}

func (s *PgStorage) migrate(ctx context.Context) error {
	db, err := goose.OpenDBWithDriver("postgres", s.databaseDSN)
	if err != nil {
		return fmt.Errorf("open db to migrate: %w", err)
	}
	if err = goose.RunContext(ctx, "up", db, "migrations"); err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}
	return nil
}
