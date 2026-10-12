package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gobwas/glob"
	"github.com/quackitsquinn/moss/internal/db"
	"github.com/quackitsquinn/moss/internal/server"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:     os.Args[0],
		Commands: Commands(),
		Usage:    "Moss is a tool for managing and querying large datasets.",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db",
				Usage:    "Root directory to the database. Optionally specified with the MOSS_DATA_DIRECTORY environment variable.",
				Required: true,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func Commands() []*cli.Command {
	generate := GenerateDBCommand()
	remove := RemoveDBCommand()
	server := RunServerCommand()
	return []*cli.Command{
		&generate,
		&remove,
		&server,
	}
}

func GenerateDBCommand() cli.Command {
	return cli.Command{
		Name:    "generate-db",
		Usage:   "Generate a database from a source directory",
		Aliases: []string{"gen-db"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "src",
				Usage:    "Root source directory.",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "replace",
				Usage: "Replace existing database",
			},
			&cli.StringFlag{
				Name:  "glob",
				Usage: "Only links files that match the given glob",
				Value: "*",
			},
		},
		Action: GenerateDBAction,
	}
}

func RemoveDBCommand() cli.Command {
	return cli.Command{
		Name:    "remove-db",
		Usage:   "Remove a database",
		Aliases: []string{"rm-db"},
		Action:  RemoveDBAction,
	}
}

func RemoveDBAction(ctx context.Context, c *cli.Command) error {
	rootDstPtr, err := ReadCurrentDBPath(c)
	if err != nil {
		return err
	}
	rootDst := *rootDstPtr

	if err = db.RemoveDB(rootDst); !os.IsNotExist(err) && err != nil {
		return fmt.Errorf("error removing existing database: %w", err)
	}

	fmt.Printf("Database at %s removed successfully\n", rootDst)
	return nil
}

func RunServerCommand() cli.Command {
	return cli.Command{
		Name:    "run-server",
		Usage:   "Run the moss server",
		Aliases: []string{"server"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host_addr",
				Usage: "Port to listen on",
				Value: ":8080",
			},
		},
		Action: RunServerAction,
	}
}

func RunServerAction(ctx context.Context, c *cli.Command) error {
	hostAddr := c.String("host_addr")
	db, err := LoadCurrentDB(c)
	if err != nil {
		return err
	}

	cfg := server.Config{
		HostAddr: hostAddr,
		DB:       *db,
	}

	if err := server.RunServer(cfg); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	return nil
}

const DBEnvVar = "MOSS_DATA_DIR"

func ReadCurrentDBPath(c *cli.Command) (*string, error) {
	dbPath := c.String("db")
	if c.String("db") != "" {
		return &dbPath, nil
	}
	env := os.Getenv(DBEnvVar)
	if env != "" {
		return &env, nil
	}
	return nil, fmt.Errorf("no database path provided. Please specify a path with the --db flag or set the MOSS_DATA_DIR environment variable")
}

func LoadCurrentDB(c *cli.Command) (*db.DB, error) {
	rootDstPtr, err := ReadCurrentDBPath(c)
	if err != nil {
		return nil, err
	}
	rootDst := *rootDstPtr

	db, err := db.LoadDB(rootDst)
	if err != nil {
		return nil, fmt.Errorf("failed to load database: %w", err)
	}
	return db, nil
}

func GenerateDBAction(ctx context.Context, c *cli.Command) error {
	rootSrc := c.String("src")
	rootDstPtr, err := ReadCurrentDBPath(c)
	if err != nil {
		return err
	}
	rootDst := *rootDstPtr
	shouldReplaceOld := c.Bool("replace")
	srcGlob := c.String("glob")

	if shouldReplaceOld {
		if err = db.RemoveDB(rootDst); !os.IsNotExist(err) && err != nil {
			return fmt.Errorf("error removing existing database: %w", err)
		}
	} else if _, err = os.Stat(rootDst); err == nil {
		return fmt.Errorf("destination directory already exists: %s", rootDst)
	}

	glob, err := glob.Compile(srcGlob)
	if err != nil {
		return fmt.Errorf("failed to compile given glob: %w", err)
	}

	fmt.Printf("Generating database at %s from source %s\n", rootDst, rootSrc)
	if err := db.GenerateDB(rootSrc, rootDst, *glob); err != nil {
		return fmt.Errorf("error generating database: %w", err)
	}

	return nil
}
