package main

import (
	"flag"
	"log/slog"
	"net/http"

	"github.com/quackitsquinn/moss/internal/db"
)

func main() {
	var db_path = flag.String("db", "", "Path to the database")
	var addr = flag.String("host_addr", ":8080", "Port to listen on")
	flag.Parse()

	if *db_path == "" {
		flag.Usage()
		return
	}

	db, err := db.LoadDB(*db_path)
	if err != nil {
		slog.Error("Failed to load database", "error", err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		doRandomFile(w, r, db)
	})
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}

func doRandomFile(w http.ResponseWriter, r *http.Request, db *db.DB) {
	w.Header().Add("Content-Type", "image/png")
	w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")

	selectedFile, err := db.SelectFile()
	if err != nil {
		slog.Error("Failed to select file", "error", err)
		http.Error(w, "Failed to select file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Serving file", "file", *selectedFile)
	http.ServeFile(w, r, *selectedFile)
}
