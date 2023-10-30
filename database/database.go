package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound       = errors.New("entry not found")
	ErrDuplicateEmail = errors.New("email already exists")

	db *sqlx.DB
)

func Connect(host, username, password, database string) error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		username, password,
		host, database,
	)
	var err error
	db, err = sqlx.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetMaxOpenConns(10)
	return nil
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
}
