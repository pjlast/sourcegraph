// Package dbconn provides functionality to connect to our DB and migrate it.
//
// Most services should connect to the frontend for DB access instead, using
// api.InternalClient.
package dbconn

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

var (
	defaultDataSource      = env.Get("PGDATASOURCE", "", "Default dataSource to pass to Postgres. See https://pkg.go.dev/github.com/jackc/pgx for more information.")
	defaultApplicationName = env.Get("PGAPPLICATIONNAME", "sourcegraph", "The value of application_name appended to dataSource")
	// Ensure all time instances have their timezones set to UTC.
	// https://github.com/golang/go/blob/7eb31d999cf2769deb0e7bdcafc30e18f52ceb48/src/time/zoneinfo_unix.go#L29-L34
	_ = env.Ensure("TZ", "UTC", "timezone used by time instances")
)

// Opts contain arguments passed to database connection initialisation functions.
type Opts struct {
	// DSN (data source name) is a URI like string containing all data needed to connect to the database.
	DSN string

	// DBName is used only for Prometheus metrics instead of whatever actual database name is set in DSN.
	// This is needed because in our dev environment we use a single physical database (and DSN) for all our different
	// logical databases.
	DBName string

	// AppName overrides the application_name in the DSN. This separate parameter is needed
	// because we have multiple apps connecting to the same database, but have a single shared DSN configured.
	AppName string
}

// New connects to the given data source and returns the handle.
//
// dbname is used for its Prometheus label value instead of whatever actual value is set in dataSource.
// This is needed because in our dev environment we use a single physical database (and DSN) for all our different
// logical databases. app, however is set as the application_name in the connection string. This is needed
// because we have multiple apps connecting to the same database, but have a single shared DSN.
//
// Note: github.com/jackc/pgx parses the environment as well. This function will
// also use the value of PGDATASOURCE if supplied and dataSource is the empty
// string.
func New(opts Opts) (*sql.DB, error) {
	cfg, err := buildConfig(opts.DSN, opts.AppName)
	if err != nil {
		return nil, err
	}

	db, err := newWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	prometheus.MustRegister(newMetricsCollector(db, opts.DBName, opts.AppName))
	configureConnectionPool(db)

	return db, nil
}

// NewRaw connects to the given data source and returns the handle.
//
// Prefer to call New as it also configures a connection pool and metrics.
// Use this method only in internal utilities (such as schemadoc).
func NewRaw(dataSource string) (*sql.DB, error) {
	cfg, err := buildConfig(dataSource, "")
	if err != nil {
		return nil, err
	}
	return newWithConfig(cfg)
}

// buildConfig takes either a Postgres connection string or connection URI,
// parses it, and returns a config with additional parameters.
func buildConfig(dataSource, app string) (*pgx.ConnConfig, error) {
	if dataSource == "" {
		dataSource = defaultDataSource
	}

	if app == "" {
		app = defaultApplicationName
	}

	cfg, err := pgx.ParseConfig(dataSource)
	if err != nil {
		return nil, err
	}

	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}

	// pgx doesn't support dbname so we emulate it
	if dbname, ok := cfg.RuntimeParams["dbname"]; ok {
		cfg.Database = dbname
		delete(cfg.RuntimeParams, "dbname")
	}

	// pgx doesn't support fallback_application_name so we emulate it
	// by checking if application_name is set and setting a default
	// value if not.
	if _, ok := cfg.RuntimeParams["application_name"]; !ok {
		cfg.RuntimeParams["application_name"] = app
	}

	// Force PostgreSQL session timezone to UTC.
	// pgx doesn't support the PGTZ environment variable, we need to pass
	// that information in the configuration instead.
	tz := "UTC"
	if v, ok := os.LookupEnv("PGTZ"); ok && v != "UTC" && v != "utc" {
		log15.Warn("Ignoring PGTZ environment variable; using PGTZ=UTC.", "ignoredPGTZ", v)
	}
	// We set the environment variable to PGTZ to avoid bad surprises if and when
	// it will be supported by the driver.
	if err := os.Setenv("PGTZ", "UTC"); err != nil {
		return nil, errors.Wrap(err, "Error setting PGTZ=UTC")
	}
	cfg.RuntimeParams["timezone"] = tz

	// Ensure the TZ environment variable is set so that times are parsed correctly.
	if _, ok := os.LookupEnv("TZ"); !ok {
		log15.Warn("TZ environment variable not defined; using TZ=''.")
		if err := os.Setenv("TZ", ""); err != nil {
			return nil, errors.Wrap(err, "Error setting TZ=''")
		}
	}

	return cfg, nil
}

// configureConnectionPool sets reasonable sizes on the built in DB queue. By
// default the connection pool is unbounded, which leads to the error `pq:
// sorry too many clients already`.
func configureConnectionPool(db *sql.DB) {
	var err error
	maxOpen := 30
	if e := os.Getenv("SRC_PGSQL_MAX_OPEN"); e != "" {
		maxOpen, err = strconv.Atoi(e)
		if err != nil {
			log.Fatalf("SRC_PGSQL_MAX_OPEN is not an int: %s", e)
		}
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxIdleTime(time.Minute)
}
