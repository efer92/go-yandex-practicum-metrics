package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/efer92/go-yandex-practicum-metrics/internal/config/db"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/retry"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgerrcode"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, dsn string) (*DBStorage, error) {
	conn, err := db.Connect(ctx, db.DefaultConfig(dsn))
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DBStorage{db: conn}, nil
}

func isRetriablePgError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	return false
}

func (s *DBStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}

func (s *DBStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return retry.Do(func() error {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO metrics (id, mtype, value)
			VALUES ($1, $2, $3)
			ON CONFLICT (id, mtype) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`, name, model.Gauge, value)
		return err
	}, isRetriablePgError)
}

func (s *DBStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return retry.Do(func() error {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, $2, $3)
			ON CONFLICT (id, mtype) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta, updated_at = NOW()
		`, name, model.Counter, value)
		return err
	}, isRetriablePgError)
}

func (s *DBStorage) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	return retry.Do(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		for _, m := range metrics {
			switch m.MType {
			case model.Gauge:
				if m.Value == nil {
					continue
				}
				_, err = tx.ExecContext(ctx, `
					INSERT INTO metrics (id, mtype, value)
					VALUES ($1, $2, $3)
					ON CONFLICT (id, mtype) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
				`, m.ID, model.Gauge, *m.Value)
			case model.Counter:
				if m.Delta == nil {
					continue
				}
				_, err = tx.ExecContext(ctx, `
					INSERT INTO metrics (id, mtype, delta)
					VALUES ($1, $2, $3)
					ON CONFLICT (id, mtype) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta, updated_at = NOW()
				`, m.ID, model.Counter, *m.Delta)
			}
			if err != nil {
				return fmt.Errorf("exec metric %s: %w", m.ID, err)
			}
		}

		return tx.Commit()
	}, isRetriablePgError)
}

func (s *DBStorage) GetMetric(ctx context.Context, name string, mType string) (*model.Metrics, bool) {
	var result *model.Metrics

	err := retry.Do(func() error {
		row := s.db.QueryRowContext(ctx, `
			SELECT id, mtype, delta, value FROM metrics WHERE id = $1 AND mtype = $2
		`, name, mType)

		m := &model.Metrics{}
		var delta sql.NullInt64
		var value sql.NullFloat64
		if err := row.Scan(&m.ID, &m.MType, &delta, &value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if delta.Valid {
			m.Delta = &delta.Int64
		}
		if value.Valid {
			m.Value = &value.Float64
		}
		result = m
		return nil
	}, isRetriablePgError)

	if err != nil || result == nil {
		return nil, false
	}
	return result, true
}

func (s *DBStorage) GetValue(ctx context.Context, mType string, name string) (string, bool) {
	m, ok := s.GetMetric(ctx, name, mType)
	if !ok {
		return "", false
	}
	switch mType {
	case model.Gauge:
		if m.Value != nil {
			return fmt.Sprintf("%g", *m.Value), true
		}
	case model.Counter:
		if m.Delta != nil {
			return fmt.Sprintf("%d", *m.Delta), true
		}
	}
	return "", false
}

func (s *DBStorage) GetAllMetrics(ctx context.Context) map[string]string {
	var result map[string]string

	retry.Do(func() error {
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, mtype, delta, value FROM metrics
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		result = make(map[string]string)
		for rows.Next() {
			var id, mtype string
			var delta sql.NullInt64
			var value sql.NullFloat64
			if err := rows.Scan(&id, &mtype, &delta, &value); err != nil {
				continue
			}
			switch mtype {
			case model.Gauge:
				if value.Valid {
					result["gauge/"+id] = fmt.Sprintf("%g", value.Float64)
				}
			case model.Counter:
				if delta.Valid {
					result["counter/"+id] = fmt.Sprintf("%d", delta.Int64)
				}
			}
		}
		return rows.Err()
	}, isRetriablePgError)

	return result
}
