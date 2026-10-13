package server

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/quackitsquinn/moss/internal/db"
)

// Config holds the configuration for the server.
type Config struct {
	Server ServerConfig
	DB     DBConfig
}
type ServerConfig struct {
	HostAddr     string
	MaxBatchSize uint32
}

type DBConfig struct {
	Path          string
	ContentType   string
	FileExtension string
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			HostAddr:     "localhost:8080",
			MaxBatchSize: 64,
		},
		DB: DBConfig{
			Path:          "./db",
			ContentType:   "image/png",
			FileExtension: ".png",
		},
	}
}

// RunServer starts the HTTP server with the given configurations.
func RunServer(config Config) error {
	db, err := db.LoadDB(config.DB.Path)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		doRandomFile(w, r, &config, db)
	})
	err = http.ListenAndServe(config.Server.HostAddr, nil)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

func doRandomFile(w http.ResponseWriter, r *http.Request, config *Config, db *db.DB) {
	w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")

	n_str := r.URL.Query().Get("n")
	var n uint32
	if n_str == "" {
		n = 1
	} else {
		n64, err := strconv.ParseUint(n_str, 10, 32)
		if err != nil {
			slog.Error("Invalid 'n' parameter", "error", err)
			http.Error(w, "Invalid 'n' parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
		n = uint32(n64)
	}

	if n == 1 {
		w.Header().Set("Content-Type", config.DB.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/zip")
	}

	if config.Server.MaxBatchSize > 0 && n > config.Server.MaxBatchSize {
		slog.Error("Requested batch size exceeds maximum", "requested", n, "max", config.Server.MaxBatchSize)
		http.Error(w, "Requested batch size exceeds maximum: "+strconv.Itoa(int(config.Server.MaxBatchSize)), http.StatusBadRequest)
		return
	}

	var z *zip.Writer
	if n > 1 {
		slog.Info("Serving batch of files", "count", n)
		zip := zip.NewWriter(w)
		z = zip
		defer func() {
			if err := zip.Close(); err != nil {
				slog.Error("Failed to close zip writer", "error", err)
			}
		}()
	}
	var lastFile *string
	for i := range n {
		selectedFile, err := db.SelectFile()
		if err != nil {

			slog.Error("Failed to select file", "error", err)
			http.Error(w, "Failed to select file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		lastFile = selectedFile
		if z != nil {
			file, err := os.Open(*selectedFile)
			if err != nil {
				slog.Error("Failed to open file", "file", *selectedFile, "error", err)
				http.Error(w, "Failed to open file: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer file.Close()

			zipfile, err := z.Create(fmt.Sprint(i, config.DB.FileExtension))
			if err != nil {
				z.Close()
				slog.Error("Failed to create zip entry", "error", err)
				http.Error(w, "Failed to create zip entry: "+err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = io.Copy(zipfile, file)
			if err != nil {
				z.Close()
				slog.Error("Failed to write file to zip", "error", err)
				http.Error(w, "Failed to write file to zip: "+err.Error(), http.StatusInternalServerError)
				return
			}

		}
	}

	if z != nil {
		slog.Info("Successfully served batch of files", "count", n, "last_file", lastFile)
		if err := z.Close(); err != nil {
			slog.Error("Failed to close zip writer", "error", err)
			http.Error(w, "Failed to close zip writer: "+err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	if lastFile != nil {
		http.ServeFile(w, r, *lastFile)
		return
	}

	http.Error(w, "No Image Found", http.StatusNotFound)
}
