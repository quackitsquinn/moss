package db

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
)

// GenerateDB generates a database at the specified rootDst path based on the files found in the rootSrc path.
func GenerateDB(rootSrc, rootDst string) error {
	var builder = newDBBuilder()

	fmt.Printf("Generating database at %s from source %s\n", rootDst, rootSrc)

	if err := os.MkdirAll(path.Join(rootDst, "0"), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", rootDst, err)
	}

	err := filepath.WalkDir(rootSrc, func(
		path string,
		d fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		err = builder.submitFile(path, rootDst)
		if err != nil {
			return fmt.Errorf("failed to submit file %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	return nil
}

// RemoveDB removes the database at the specified rootDst path.
func RemoveDB(rootDst string) error {
	if _, err := os.Stat(rootDst); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(rootDst)
}

const bitsPerLevel = 4
const maxFilesPerDir = (1 << bitsPerLevel) - 1      // 16
const maxSubfoldersPerDir = (1 << bitsPerLevel) - 1 // 16

type dbBuilder struct {
	current_path []uint8
	file_idx     uint8
}

func newDBBuilder() *dbBuilder {
	return &dbBuilder{
		current_path: []uint8{0},
		file_idx:     0,
	}
}

func formatPath(dbBase string, intpath []uint8) string {
	var formatted = dbBase

	for _, level := range intpath {
		formatted = path.Join(formatted, fmt.Sprintf("%x", level))
	}

	return formatted
}

// bumpFolder increments the current path to the next available folder in the database structure.
func (b *dbBuilder) bumpFolder(dbBase string) (*string, error) {
	for path_idx_back := len(b.current_path) - 1; path_idx_back >= 0; path_idx_back-- {
		idx := b.current_path[path_idx_back]
		if idx < maxSubfoldersPerDir {
			b.current_path[path_idx_back]++
			var newPath = formatPath(dbBase, b.current_path)
			return &newPath, nil
		}

		b.current_path[path_idx_back] = 0
	}

	b.current_path = slices.Repeat([]uint8{0}, len(b.current_path)+1)

	var newPath = formatPath(dbBase, b.current_path)
	return &newPath, nil
}

// nextFolder creates the next folder in the database structure.
func (b *dbBuilder) nextFolder(dbBase string) error {
	newPath, err := b.bumpFolder(dbBase)
	if err != nil {
		return fmt.Errorf("failed to bump folder: %w", err)
	}

	if err := os.MkdirAll(*newPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create new folder %s: %w", *newPath, err)
	}

	return nil
}

// bump increments the current path to the next available path in the database structure.
func (b *dbBuilder) bump(dbBase string) error {
	if b.file_idx < maxFilesPerDir {
		b.file_idx++
		return nil
	}
	b.file_idx = 0
	b.nextFolder(dbBase)
	return nil
}

// currentFile returns the path to the current file in the database structure.
func (b *dbBuilder) currentFile(dbBase string) string {
	return path.Join(formatPath(dbBase, b.current_path), fmt.Sprintf("%x", b.file_idx))
}

// parentFolder returns the path to the parent folder of the current path in the database structure, returning the base path if the current path is empty.
func (b *dbBuilder) parentFolder(dbBase string) string {
	if len(b.current_path) < 1 {
		return dbBase
	}
	return formatPath(dbBase, b.current_path[:len(b.current_path)-1])
}

// submitFile creates a symlink for the specified file in the database structure and increments the current path to the next available path.
func (b *dbBuilder) submitFile(filePath, dbBase string) error {
	fmt.Printf("Submitting file: %s\n", filePath)
	if err := os.Symlink(filePath, b.currentFile(dbBase)+path.Ext(filePath)); err != nil {
		return fmt.Errorf("failed to create symlink for file %s: %w", filePath, err)
	}
	if err := b.bump(dbBase); err != nil {
		return fmt.Errorf("failed to bump path for file %s: %w", filePath, err)
	}

	return nil
}
