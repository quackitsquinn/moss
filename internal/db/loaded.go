package db

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
)

// DB represents the database structure, containing the largest folder path, the current file index, and the base path of the database.
type DB struct {
	highest_folder []uint8
	file_idx       uint8
	base           string
}

func LoadDB(dbBase string) (*DB, error) {
	largest_folder, file_idx, err := readDB(dbBase)
	if err != nil {
		return nil, fmt.Errorf("failed to load database: %w", err)
	}

	return &DB{
		highest_folder: largest_folder,
		file_idx:       file_idx,
		base:           dbBase,
	}, nil
}

func (db *DB) Base() string {
	return db.base
}

// FilePrefix is the prefix that is appended to file names when constructing the full path to a file.
const FilePrefix = "."

func (db *DB) FilePath(indices []uint8) (*string, error) {
	if len(indices) < 1 {
		return nil, errors.New("indices cannot be empty")
	}

	if len(indices) > len(db.highest_folder)+1 {
		return nil, errors.New("indices exceed the depth of the database")
	}

	file_idx := indices[len(indices)-1]
	path := formatPath(db.base, indices[:len(indices)-1], &file_idx)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist at path: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file at path %s: %w", path, err)
	}
	return &path, nil
}

func (db *DB) genNumberOfIndices() int {
	exp := math.Min(rand.ExpFloat64(), 1.0)
	return int(exp * float64(len(db.highest_folder)-1))
}

func (db *DB) SelectFile() (*string, error) {
	if len(db.highest_folder) == 0 {
		return nil, errors.New("database is empty")
	}

	nIndices := db.genNumberOfIndices() + 2
	path := make([]uint8, nIndices)
	for i := range nIndices {
		if i == len(db.highest_folder)-2 {
			path[i] = uint8(rand.IntN(int(db.highest_folder[len(db.highest_folder)-1])))
			continue
		}

		if nIndices < 3 {
			path[i] = uint8(rand.IntN(maxSubfoldersPerDir + 1))
			continue
		}

		path[i] = uint8(rand.IntN(int(db.highest_folder[len(db.highest_folder)-1]) + 1))
	}

	file, err := db.FilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to select file: %w", err)
	}

	return file, nil
}
