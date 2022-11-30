package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"
)

type Migration struct {
	ID        int64  `db:"id"`
	Name      string `db:"name"`
	CreatedAt int64  `db:"created_at"`
}

const noMigrationsTablePqError = "pq: relation \"migration\" does not exist"

//go:embed scheme
var scheme embed.FS

func (s *storage) executeMigrations(ctx context.Context, db *sqlx.DB) error {

	var rows []Migration
	if err := db.SelectContext(ctx, &rows, "SELECT * FROM migration"); err != nil && err.Error() != noMigrationsTablePqError {
		return err
	}

	appliedMigrations := make(map[string]struct{})
	for _, row := range rows {
		appliedMigrations[row.Name] = struct{}{}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %s", err)
	}

	defer func() {
		tx.Rollback()
	}()

	currentlyExecutedMigrations := []string{}
	commentsRegExp := regexp.MustCompile(`/\*.*\*/`)

	if err := fs.WalkDir(scheme, "scheme", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "scheme" {
			return nil
		}

		_, fileName := filepath.Split(path)
		if _, ok := appliedMigrations[fileName]; ok {
			return nil
		}

		fileContent, err := fs.ReadFile(scheme, path)
		if err != nil {
			return err
		}

		sql := commentsRegExp.ReplaceAllString(string(fileContent), "")

		if _, err = tx.Exec(sql); err != nil {
			return err
		}

		currentlyExecutedMigrations = append(currentlyExecutedMigrations, fileName)

		return nil

	}); err != nil {
		return fmt.Errorf("failed to apply migartions: %s", err)
	}

	now := time.Now().UnixNano()
	for _, executedMigration := range currentlyExecutedMigrations {
		if _, err := tx.ExecContext(ctx, db.Rebind("INSERT INTO migration (name, created_at) VALUES(?, ?);"), executedMigration, now); err != nil {
			return fmt.Errorf("failed to insert executed migration: %s", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations transaction: %s", err)
	}

	return nil
}
