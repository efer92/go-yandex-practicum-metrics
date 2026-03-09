package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/efer92/go-yandex-practicum-metrics/internal/config/db"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, dsn string) (*DBStorage, error) {
	conn, err := db.Connect(ctx, db.DefaultConfig(dsn))
	if err != nil {
		return nil, err
	}
	s := &DBStorage{db: conn}
	if err := s.migrate(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *DBStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}

func (s *DBStorage) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS metrics (
			id    TEXT             NOT NULL,
			mtype TEXT             NOT NULL,
			delta BIGINT,
			value DOUBLE PRECISION,
			PRIMARY KEY (id, mtype)
		)
	`)
	return err
}

// --- реализация интерфейса Storage ---

func (s *DBStorage) UpdateGauge(name string, value float64) error {
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO metrics (id, mtype, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype) DO UPDATE SET value = EXCLUDED.value
	`, name, model.Gauge, value)
	return err
}

func (s *DBStorage) UpdateCounter(name string, value int64) error {
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
	`, name, model.Counter, value)
	return err
}

func (s *DBStorage) GetMetric(name string, mType string) (*model.Metrics, bool) {
	row := s.db.QueryRowContext(context.Background(), `
		SELECT id, mtype, delta, value FROM metrics WHERE id = $1 AND mtype = $2
	`, name, mType)

	m := &model.Metrics{}
	var delta sql.NullInt64
	var value sql.NullFloat64
	if err := row.Scan(&m.ID, &m.MType, &delta, &value); err != nil {
		return nil, false
	}
	if delta.Valid {
		m.Delta = &delta.Int64
	}
	if value.Valid {
		m.Value = &value.Float64
	}
	return m, true
}

func (s *DBStorage) GetValue(mType string, name string) (string, bool) {
	m, ok := s.GetMetric(name, mType)
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

func (s *DBStorage) GetAllMetrics() map[string]string {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id, mtype, delta, value FROM metrics
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make(map[string]string)
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
	if err := rows.Err(); err != nil {
		return nil
	}
	return result
}
