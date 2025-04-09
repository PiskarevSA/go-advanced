package pgstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const retryCount = 3

type PgStorage struct {
	databaseDSN string
	pool        *pgxpool.Pool
}

func New(ctx context.Context, databaseDSN string) (*PgStorage, error) {
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

func (s *PgStorage) GetMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	switch metric.Type {
	case entities.MetricTypeGauge:
		query := "select value from gauge where name = $1"
		var value entities.Gauge

		doQueries := func(tx pgx.Tx) error {
			row := tx.QueryRow(ctx, query, metric.Name)
			return row.Scan(&value)
		}

		err := doTransactionWithRetries(ctx, s.pool, doQueries)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		} else if err != nil {
			return nil, entities.NewInternalError("sql query error", err)
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

		doQueries := func(tx pgx.Tx) error {
			row := tx.QueryRow(ctx, query, metric.Name)
			return row.Scan(&value)
		}

		err := doTransactionWithRetries(ctx, s.pool, doQueries)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		} else if err != nil {
			return nil, entities.NewInternalError("sql query error", err)
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
	switch metric.Type {
	case entities.MetricTypeGauge:
		query := `
			insert into gauge (name, value)
			values ($1, $2)
			on conflict(name)
			do update set
			  value = excluded.value
			returning value`
		var value entities.Gauge

		doQueries := func(tx pgx.Tx) error {
			row := tx.QueryRow(ctx, query, metric.Name, metric.Value)
			return row.Scan(&value)
		}

		err := doTransactionWithRetries(ctx, s.pool, doQueries)
		if err != nil {
			return nil, entities.NewInternalError("sql query error", err)
		}

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		query := `
			insert into counter (name, value)
			values ($1, $2)
			on conflict(name)
			do update set
			  value = counter.value + excluded.value
			returning value`
		var value entities.Counter

		doQueries := func(tx pgx.Tx) error {
			row := tx.QueryRow(ctx, query, metric.Name, metric.Delta)
			return row.Scan(&value)
		}

		err := doTransactionWithRetries(ctx, s.pool, doQueries)
		if err != nil {
			return nil, entities.NewInternalError("sql query error", err)
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
	var result []entities.Metric
	var err error
	doQueries := func(tx pgx.Tx) error {
		result = make([]entities.Metric, 0)
		for i, metric := range metrics {
			switch metric.Type {
			case entities.MetricTypeGauge:
				query := `
				insert into gauge (name, value)
				values ($1, $2)
				on conflict(name)
				do update set
				  value = excluded.value
				returning value`
				var value entities.Gauge
				row := tx.QueryRow(ctx, query, metric.Name, metric.Value)
				err = row.Scan(&value)
				if err != nil {
					return entities.NewInternalError(
						fmt.Sprintf("metric[%v]: sql query error", i), err)
				}

				updatedMetric := entities.Metric{
					Type:  metric.Type,
					Name:  metric.Name,
					Value: value,
					Delta: 0,
				}
				result = append(result, updatedMetric)

			case entities.MetricTypeCounter:
				query := `
				insert into counter (name, value)
				values ($1, $2)
				on conflict(name)
				do update set
				value = counter.value + excluded.value
				returning value`
				var value entities.Counter
				row := tx.QueryRow(ctx, query, metric.Name, metric.Delta)
				err = row.Scan(&value)
				if err != nil {
					return entities.NewInternalError(
						fmt.Sprintf("metric[%v]: sql query error", i), err)
				}

				updatedMetric := entities.Metric{
					Type:  metric.Type,
					Name:  metric.Name,
					Value: 0,
					Delta: value,
				}
				result = append(result, updatedMetric)
			default:
				return entities.NewInternalError(
					fmt.Sprintf("metric[%v]: unexpected internal metric type: %v",
						i, metric.Type.String()), nil)
			}
		}
		return nil
	}
	err = doTransactionWithRetries(ctx, s.pool, doQueries)
	return result, err
}

func (s *PgStorage) GetMetricsByTypes(ctx context.Context,
	gauge map[entities.MetricName]entities.Gauge,
	counter map[entities.MetricName]entities.Counter,
) error {
	doQueries := func(tx pgx.Tx) error {
		var err error

		func() {
			query := "select name, value from gauge"
			var name entities.MetricName
			var gaugeValue entities.Gauge
			var rows pgx.Rows
			rows, err = tx.Query(ctx, query)
			if err != nil {
				err = entities.NewInternalError("sql query error", err)
				return
			}
			defer rows.Close()

			for rows.Next() {
				if err = rows.Scan(&name, &gaugeValue); err != nil {
					err = entities.NewInternalError("sql query error", err)
					return
				}
				gauge[name] = gaugeValue
			}
			if rows.Err() != nil {
				err = entities.NewInternalError("sql query error", rows.Err())
				return
			}
		}()
		if err != nil {
			return err
		}

		func() {
			query := "select name, value from counter"
			var name entities.MetricName
			var counterValue entities.Counter
			var rows pgx.Rows
			rows, err = tx.Query(ctx, query)
			if err != nil {
				err = entities.NewInternalError("sql query error", err)
				return
			}
			defer rows.Close()

			for rows.Next() {
				if err = rows.Scan(&name, &counterValue); err != nil {
					err = entities.NewInternalError("sql query error", err)
					return
				}
				counter[name] = counterValue
			}
			if rows.Err() != nil {
				err = entities.NewInternalError("sql query error", rows.Err())
				return
			}
		}()
		return err
	}
	return doTransactionWithRetries(ctx, s.pool, doQueries)
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
