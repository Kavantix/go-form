package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxStdlib "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var (
	pool customPool
)

type customPool struct {
	rawPool *pgxpool.Pool
}

func (c customPool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	span := startQuerySpan(ctx, sql, false)
	defer span.Finish()
	result, err := c.rawPool.Exec(ctx, sql, args...)
	if errors.Is(err, context.DeadlineExceeded) {
		span.Status = sentry.SpanStatusDeadlineExceeded
	}
	return result, err
}
func (c customPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	span := startQuerySpan(ctx, sql, false)
	defer span.Finish()
	result, err := c.rawPool.Query(ctx, sql, args...)
	if errors.Is(err, context.DeadlineExceeded) {
		span.Status = sentry.SpanStatusDeadlineExceeded
	}
	return result, err
}
func (c customPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	span := startQuerySpan(ctx, sql, false)
	defer span.Finish()
	return c.rawPool.QueryRow(ctx, sql, args...)
}

func (c customPool) Close() {
	log.Println("Closing database connection pool")
	c.rawPool.Close()
}

type CountResult struct {
	Count int `db:"count"`
}

type sentryTracer struct {
}

func startQuerySpan(ctx context.Context, sql string, isExecute bool) *sentry.Span {
	var span *sentry.Span
	if isExecute {
		fmt.Printf("Starting execute query: %v,\n", time.Now())
		span = sentry.StartSpan(ctx, "db.sql.execute")
	} else {
		fmt.Printf("Starting query: %v,\n", time.Now())
		span = sentry.StartSpan(ctx, "db.sql.query")
	}
	span.SetData("db.system", "postgresql")
	span.Description = sql
	return span
}

func (s sentryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	span := startQuerySpan(ctx, data.SQL, true)
	return context.WithValue(ctx, "querySpan", span)
}

func (s sentryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := ctx.Value("querySpan").(*sentry.Span)
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		if errors.Is(data.Err, context.DeadlineExceeded) {
			span.Status = sentry.SpanStatusDeadlineExceeded
		} else {
			span.Status = sentry.SpanStatusInternalError
		}
	}
	fmt.Printf("Finish query: %v\n", time.Now())
	span.Finish()
}

func Connect(host, port, username, password, database, sslmode string) (*Queries, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		username, password,
		host, port, database,
		sslmode,
	)
	var err error
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create database config: %w", err)
	}
	config.ConnConfig.Tracer = sentryTracer{}
	config.BeforeClose = func(c *pgx.Conn) {
		fmt.Println("closing connection")
	}
	config.MaxConns = 100
	config.MinConns = 10
	rawPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	pool = customPool{rawPool: rawPool}
	queries := New(pool)
	return queries, nil
}

type DebugOptions struct {
	IncludeValues bool
}

func Debug(opts DebugOptions) {
	db := sqlx.NewDb(pgxStdlib.OpenDB(*pool.rawPool.Config().ConnConfig), "postgres")
	defer db.Close()
	tables := []struct {
		Name string `db:"tablename"`
	}{}
	err := db.Select(&tables, "SELECT tablename FROM pg_catalog.pg_tables where schemaname = 'public'")
	if err != nil {
		log.Fatal(err)
	}
	for _, table := range tables {
		fmt.Printf("Table: %s\n", table.Name)
		rows, err := db.Queryx(fmt.Sprintf("select * from \"%s\" limit 3", table.Name))
		if err != nil {
			log.Fatal(err)
		}
		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			log.Fatal(err)
		}
		for _, column := range columnTypes {
			fmt.Printf(" %s (%s)\n", column.Name(), column.ScanType())
		}
		row := map[string]any{}
		if opts.IncludeValues {
			fmt.Println("  [")
			for rows.Next() {
				err := rows.MapScan(row)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("    {")
				for _, column := range columnTypes {
					fmt.Printf("     %s: %v\n", column.Name(), row[column.Name()])

				}
				fmt.Println("    }")
			}
			fmt.Println("  ]")
		}
	}
}

func Close() {
	pool.Close()
}
