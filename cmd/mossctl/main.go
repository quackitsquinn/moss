package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gobwas/glob"
	"github.com/pelletier/go-toml"
	"github.com/quackitsquinn/moss/internal/db"
	"github.com/quackitsquinn/moss/internal/server"
	"github.com/urfave/cli/v3"
)

// main is the entry point for the mossctl command-line tool.
func main() {
	cmd := &cli.Command{
		Name:     os.Args[0],
		Commands: Commands(),
		Usage:    "Moss is a tool for managing and querying large datasets.",
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

// DbFlag returns a cli.Flag for specifying the database path.
func DbFlag() cli.Flag {
	return &cli.StringFlag{
		Name:     "db",
		Usage:    "Root directory to the database. Optionally specified with the MOSS_DATA_DIRECTORY environment variable.",
		Required: true,
	}
}

// Commands returns the list of CLI commands for the application.
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

// GenerateDBCommand returns a cli.Command for generating a database from a source directory.
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
			DbFlag(),
		},
		Action: GenerateDBAction,
	}
}

// RemoveDBCommand returns a cli.Command for removing an existing database.
func RemoveDBCommand() cli.Command {
	return cli.Command{
		Name:    "remove-db",
		Usage:   "Remove a database",
		Aliases: []string{"rm-db"},
		Action:  RemoveDBAction,
		Flags: []cli.Flag{
			DbFlag(),
		},
	}
}

// RemoveDBAction removes the database at the specified path.
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

// RunServerCommand returns a cli.Command for running the moss server.
func RunServerCommand() cli.Command {
	return cli.Command{
		Name:  "server",
		Usage: "Run the moss server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config_file",
				Aliases:  []string{"c", "cfg", "config"},
				Usage:    "Path to the configuration file",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "run",
				Usage:  "Run the moss server",
				Action: RunServerAction,
			},
			{
				Name:   "defconfig",
				Usage:  "Generate a default configuration file",
				Action: DefConfigAction,
			},
		},
		Action: RunServerAction,
	}
}

// DefConfigAction generates a default configuration file at the specified path.
func DefConfigAction(ctx context.Context, c *cli.Command) error {
	config := server.DefaultConfig()
	configFile := c.String("config_file")

	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	enc := toml.NewEncoder(file)
	if err := enc.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config to file: %w", err)
	}

	fmt.Printf("Default configuration written to %s\n", configFile)
	return nil
}

// LoadConfig loads the server configuration from the specified TOML file.
func LoadConfig(path string) (*server.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	dec := toml.NewDecoder(file)

	cfg := server.Config{}
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &cfg, nil
}

// RunServerAction runs the moss server with the specified configuration.
func RunServerAction(ctx context.Context, c *cli.Command) error {
	cfg := c.String("config_file")
	config, err := LoadConfig(cfg)
	if err != nil {
		return err
	}

	if err := server.RunServer(*config); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	return nil
}

// DBEnvVar is the environment variable used to specify the database path.
const DBEnvVar = "MOSS_DATA_DIR"

// ReadCurrentDBPath reads the database path from the command line flag or environment variable.
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

// LoadCurrentDB loads the database from the path specified in the command line flag or environment variable.
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

// GenerateDBAction generates a database from the specified source directory and writes it to the specified destination directory.
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
