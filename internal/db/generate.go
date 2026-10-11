package db

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/gobwas/glob"
)

// GenerateDB generates a database at the specified rootDst path based on the files found in the rootSrc path.
func GenerateDB(rootSrc, relDst string, filter glob.Pattern) error {
	var builder = newDBBuilder()
	rootSrc, err := filepath.Abs(rootSrc)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for source directory %s: %w", rootSrc, err)
	}
	rootDst, err := filepath.Abs(relDst)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for destination directory %s: %w", relDst, err)
	}

	fmt.Printf("Generating database at %s from source %s\n", rootDst, rootSrc)

	if err := os.MkdirAll(path.Join(rootDst, "0"), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", rootDst, err)
	}

	err = filepath.WalkDir(rootSrc, func(
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

		if !filter.Match(path) {
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

	err = builder.build(rootDst)
	if err != nil {
		return fmt.Errorf("failed to build database: %w", err)
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

// dbBuilder is a struct that helps manage the current path and file index while generating a database structure.
type dbBuilder struct {
	current_path []uint8
	file_idx     uint8
}

// newDBBuilder creates a new dbBuilder instance with the initial path set to [0] and the file index set to 0.
func newDBBuilder() *dbBuilder {
	return &dbBuilder{
		current_path: []uint8{0},
		file_idx:     0,
	}
}

// formatPath formats the current path in the database structure as a string, using the specified dbBase as the root.
func formatPath(dbBase string, intpath []uint8, file_idx *uint8) string {
	var formatted = dbBase

	for _, level := range intpath {
		formatted = path.Join(formatted, fmt.Sprintf("%x", level))
	}

	if file_idx != nil {
		return path.Join(formatted, fmt.Sprintf(".%x", *file_idx))
	}
	return formatted
}

// bumpFolder increments the current path to the next available folder in the database structure.
func (b *dbBuilder) bumpFolder(dbBase string) (*string, error) {
	for path_idx_back := len(b.current_path) - 1; path_idx_back >= 0; path_idx_back-- {
		idx := b.current_path[path_idx_back]
		if idx < maxSubfoldersPerDir {
			b.current_path[path_idx_back]++
			var newPath = formatPath(dbBase, b.current_path, nil)
			return &newPath, nil
		}

		b.current_path[path_idx_back] = 0
	}

	b.current_path = slices.Repeat([]uint8{0}, len(b.current_path)+1)

	var newPath = formatPath(dbBase, b.current_path, nil)
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
	if err := b.nextFolder(dbBase); err != nil {
		return fmt.Errorf("failed to create next folder: %w", err)
	}
	return nil
}

// currentFile returns the path to the current file in the database structure.
func (b *dbBuilder) currentFile(dbBase string) string {
	return formatPath(dbBase, b.current_path, &b.file_idx)
}

// parentFolder returns the path to the parent folder of the current path in the database structure, returning the base path if the current path is empty.
func (b *dbBuilder) parentFolder(dbBase string) string {
	if len(b.current_path) < 1 {
		return dbBase
	}
	return formatPath(dbBase, b.current_path[:len(b.current_path)-1], nil)
}

// submitFile creates a symlink for the specified file in the database structure and increments the current path to the next available path.
func (b *dbBuilder) submitFile(filePath, dbBase string) error {
	fmt.Printf("Submitting file: %s\n", filePath)
	if err := os.Symlink(filePath, b.currentFile(dbBase)); err != nil {
		return fmt.Errorf("failed to create symlink for file %s: %w", filePath, err)
	}
	if err := b.bump(dbBase); err != nil {
		return fmt.Errorf("failed to bump path for file %s: %w", filePath, err)
	}

	return nil
}

func (b *dbBuilder) build(dbBase string) error {
	writer, err := os.Create(path.Join(dbBase, ".db"))
	if err != nil {
		return fmt.Errorf("failed to create .db file: %w", err)
	}
	defer writer.Close()

	writeByte := func(b byte) error {
		if _, err := writer.Write([]byte{b}); err != nil {
			return fmt.Errorf("failed to write to .db file: %w", err)
		}
		return nil
	}

	for _, level := range b.current_path {
		if err := writeByte(level); err != nil {
			return err
		}
	}

	if err := writeByte(b.file_idx); err != nil {
		return err
	}

	return nil
}

func readDB(dbBase string) ([]uint8, uint8, error) {
	dbFilePath := path.Join(dbBase, ".db")
	data, err := os.ReadFile(dbFilePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read .db file: %w", err)
	}

	if len(data) < 1 {
		return nil, 0, errors.New(".db file is empty or corrupted")
	}

	currentPath := data[:len(data)-1]
	fileIdx := data[len(data)-1]

	return currentPath, fileIdx, nil
}
