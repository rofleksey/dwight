package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/buntdb"
)

type DB struct {
	db *buntdb.DB
}

func NewDB() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(home, ".dwight.db")
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) GetContext(projectID string) ([]string, error) {
	var context []string
	err := d.db.View(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("project:%s:context", projectID)
		val, err := tx.Get(key)
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		context = strings.Split(val, "|")
		return nil
	})
	return context, err
}

func (d *DB) SetContext(projectID string, context []string) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("project:%s:context", projectID)
		val := strings.Join(context, "|")
		_, _, err := tx.Set(key, val, nil)
		return err
	})
}
