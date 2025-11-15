package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Photo struct {
	ID       int64  `json:"id"`
	FilePath string `json:"file_path"`
}

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) GetPhotoIDsByPaths(filePaths []string) ([]int64, error) {
	if len(filePaths) == 0 {
		return []int64{}, nil
	}

	// 构建查询语句
	query := "SELECT user_photo.id FROM photo left join user_photo on photo.id = user_photo.photo_id WHERE photo.file_path IN ("
	args := make([]interface{}, len(filePaths))
	for i, path := range filePaths {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args[i] = path
	}
	query += ")"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan photo id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return ids, nil
}
