package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type storeState struct {
	key    []byte
	values map[string]string
}

type StoredSecret struct {
	Key    string
	Cipher []byte
}

// loadStore reads the bootstrap pair from the environment only: the store
// cannot supply the database it lives in or the key that opens it.
func loadStore() error {
	key, err := ParseKey(KEKEnv, lookupEnvOnly(KEKEnv))
	if err != nil {
		return err
	}
	next := storeState{key: key}
	if key != nil {
		rows, err := fetchStore(lookupEnvOnly(DatabaseURLEnv))
		if err != nil {
			return fmt.Errorf("config store: %w", err)
		}
		if next.values, err = Resolve(key, rows); err != nil {
			return err
		}
		slog.Info("config store loaded", "values", len(next.values))
	}
	mu.Lock()
	store, loaded = next, time.Now()
	mu.Unlock()
	return nil
}

func lookupEnvOnly(key string) string {
	value, source := os.Getenv(key), SourceEnv
	if value == "" {
		source = SourceDefault
	}
	mu.Lock()
	catalog[key] = Var{Key: key, Value: value, Source: source}
	mu.Unlock()
	return value
}

func Resolve(key []byte, rows []StoredSecret) (map[string]string, error) {
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		plain, err := Open(key, r.Cipher)
		if err != nil {
			return nil, fmt.Errorf("config store: %s does not open with %s: %w", r.Key, KEKEnv, err)
		}
		out[r.Key] = plain
	}
	return out, nil
}

func fetchStore(dsn string) ([]StoredSecret, error) {
	if dsn == "" {
		return nil, errors.New(DatabaseURLEnv + " is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close(context.Background()) }()

	rows, err := conn.Query(ctx, `SELECT key, cipher FROM config_secret`)
	if err != nil {
		var pgErr *pgconn.PgError
		// migrate loads config before it has created the table.
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return nil, nil
		}
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (StoredSecret, error) {
		var s StoredSecret
		err := r.Scan(&s.Key, &s.Cipher)
		return s, err
	})
}
