package database

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Kavantix/go-form/newdatabase"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound       = newdatabase.ErrNotFound
	ErrDuplicateEmail = newdatabase.ErrDuplicateEmail

	db      *sqlx.DB
	pool    *pgxpool.Pool
	queries *newdatabase.Queries
)

type CountResult struct {
	Count int `db:"count"`
}

type sentryTracer struct {
}

func (s sentryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	span := sentry.StartSpan(ctx, "db.sql.query")
	span.SetData("db.system", "postgresql")
	span.Description = data.SQL
	return context.WithValue(ctx, "querySpan", span)
}

func (s sentryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := ctx.Value("querySpan").(*sentry.Span)
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.Status = sentry.SpanStatusInternalError
	}
	span.Finish()
}

func Connect(host, port, username, password, database, sslmode string) (*newdatabase.Queries, error) {
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
	pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	queries = newdatabase.New(pool)
	db, err = sqlx.Open("postgres", connStr)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetMaxOpenConns(10)
	return queries, nil
}

func Debug() {
	tables := []struct {
		Name string `db:"tablename"`
	}{}
	err := db.Select(&tables, "SELECT tablename FROM pg_catalog.pg_tables where schemaname = 'public'")
	if err != nil {
		log.Fatal(err)
	}
	for _, table := range tables {
		fmt.Printf("Table: %s\n", table.Name)
		rows, err := db.Queryx(fmt.Sprintf("select * from \"%s\" limit 10", table.Name))
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

func Close() {
	db.Close()
	pool.Close()
}
