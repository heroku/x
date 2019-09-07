// Package testenv exposes environment-dependent resources for testing.
package testenv

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/soveran/redisurl"
)

var skipMissingEnv = os.Getenv("TESTING_SKIP_MISSING_ENV") == "true"

// NewRedisPool returns a redis pool connected to RedisURL.
func NewRedisPool(t testing.TB) *redis.Pool {
	t.Helper()

	redisURL, err := getenv("REDIS_URL", "redis://localhost:6379")
	if err != nil {
		t.Skip(err.Error())
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := redisurl.ConnectToURL(redisURL)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
}

// OpenDatabase returns a database connection for testing the provided service.
// When the close func is called any modified data will be rolled back.
func OpenDatabase(t *testing.T, dbname string) (tx *sql.Tx, close func()) {
	t.Helper()

	dbURL, err := getenv(
		"DATABASE_URL",
		fmt.Sprintf("postgres:///%s?sslmode=disable", dbname),
	)
	if err != nil {
		t.Skip(err.Error())
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatal(err)
	}

	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	close = func() {
		if err := tx.Rollback(); err != nil {
			t.Fatal("unexpected error", err)
		}
		db.Close()
	}

	return tx, close
}

// MustDB returns a *sql.DB for dbname, or panics if the database is
// unreachable. Calling the cleanup func drops all data in the database.
func MustDB(dbname string) (db *sql.DB, cleanup func()) {
	dbURL, err := getenv(
		"DATABASE_URL",
		fmt.Sprintf("postgres:///%s?sslmode=disable", dbname),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(0)
	}

	if db, err = sql.Open("postgres", dbURL); err != nil {
		panic(err)
	}

	// Hold an exclusive lock on the schema_migrations table to serialize
	// parallel tests that use the same database.
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	if _, err := tx.Exec("LOCK TABLE schema_migrations IN ACCESS EXCLUSIVE MODE;"); err != nil {
		panic(err)
	}

	return db, func() {
		if _, err := tx.Exec(sqlTruncateTables); err != nil {
			panic(err)
		}
		if err := tx.Commit(); err != nil {
			panic(err)
		}
		if err := db.Close(); err != nil {
			panic(err)
		}
	}
}

const sqlTruncateTables = `
CREATE OR REPLACE FUNCTION truncate_tables() RETURNS void AS $$
DECLARE
    statements CURSOR FOR
        SELECT tablename FROM pg_tables
        WHERE tablename != 'schema_migrations'
          AND tableowner = session_user
          AND schemaname = 'public';
BEGIN
    FOR stmt IN statements LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(stmt.tablename) || ' CASCADE;';
    END LOOP;
END;
$$ LANGUAGE plpgsql;
SELECT truncate_tables();
`

func getenv(key, fallback string) (string, error) {
	if v := os.Getenv(key); v != "" {
		return v, nil
	}

	if skipMissingEnv {
		return "", errors.Errorf("%s not set", key)
	}

	return fallback, nil
}
