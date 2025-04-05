package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type PgStorage struct {
	databaseDSN string
	conn        *pgx.Conn
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

func (s *PgStorage) GetMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error) {
	switch metric.Type {
	case entities.MetricTypeGauge:
		row := s.conn.QueryRow(ctx, "select value from gauge where name = $1", metric.Name)
		var value entities.Gauge
		err := row.Scan(&value)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		}
		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		row := s.conn.QueryRow(ctx, "select value from counter where name = $1", metric.Name)
		var value entities.Counter
		err := row.Scan(&value)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
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
		"unexpected internal metric type: " + metric.Type.String())
}

func (s *PgStorage) UpdateMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error) {
	switch metric.Type {
	case entities.MetricTypeGauge:
		row := s.conn.QueryRow(ctx, joinLines(
			"insert into gauge (name, value)",
			"values ($1, $2)",
			"on conflict(name)",
			"do update set",
			"  value = excluded.value",
			"returning value"), metric.Name, metric.Value)
		var value entities.Gauge
		err := row.Scan(&value)
		if err != nil {
			return nil, entities.NewInternalError("sql query error: " + err.Error())
		}

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		row := s.conn.QueryRow(ctx, joinLines(
			"insert into counter (name, value)",
			"values ($1, $2)",
			"on conflict(name)",
			"do update set",
			"  value = counter.value + excluded.value",
			"returning value"), metric.Name, metric.Delta)
		var value entities.Counter
		err := row.Scan(&value)
		if err != nil {
			return nil, entities.NewInternalError("sql query error: " + err.Error())
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
		"unexpected internal metric type: " + metric.Type.String())
}

func (s *PgStorage) GetMetricsByTypes(ctx context.Context,
	gauge map[entities.MetricName]entities.Gauge,
	counter map[entities.MetricName]entities.Counter,
) error {
	gaugeRows, err := s.conn.Query(ctx, "select name, value from gauge")
	if err != nil {
		return entities.NewInternalError("sql query error: " + err.Error())
	}
	defer gaugeRows.Close()

	for gaugeRows.Next() {
		var name entities.MetricName
		var value entities.Gauge
		if err := gaugeRows.Scan(&name, &value); err != nil {
			return entities.NewInternalError("sql query error: " + err.Error())
		}
		gauge[name] = value
	}

	counterRows, err := s.conn.Query(ctx, "select name, value from counter")
	if err != nil {
		return entities.NewInternalError("sql query error: " + err.Error())
	}
	defer counterRows.Close()

	for counterRows.Next() {
		var name entities.MetricName
		var value entities.Counter
		if err := counterRows.Scan(&name, &value); err != nil {
			return entities.NewInternalError("sql query error: " + err.Error())
		}
		counter[name] = value
	}
	return nil
}

func (s *PgStorage) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

func (s *PgStorage) Close(ctx context.Context) error {
	if s.conn != nil {
		return s.conn.Close(ctx)
	}
	return nil
}

func (s *PgStorage) connect(ctx context.Context) error {
	var err error
	s.conn, err = pgx.Connect(ctx, s.databaseDSN)
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
