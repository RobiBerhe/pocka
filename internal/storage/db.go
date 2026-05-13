package storage

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
	"pocka/ent"
)

type Database struct {
	Client *ent.Client
}

func NewDatabase(databaseURL string) (*Database, error) {
	client, err := ent.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed creating schema resources: %w", err)
	}

	slog.Info("Database connection established and schema migrated.")
	return &Database{Client: client}, nil
}

func (db *Database) Close() {
	if err := db.Client.Close(); err != nil {
		slog.Error("failed to close database connection", "error", err)
	}
}
