package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gobwas/glob"
	"github.com/quackitsquinn/moss/internal/db"
	"github.com/urfave/cli/v3"
)

func main() {
	var rootSrc = flag.String("src", "", "Root source directory, supports globbing")
	var rootDst = flag.String("dst", "", "Root destination directory")
	var shouldReplaceOld = flag.Bool("replace", false, "Replace existing database")
	var srcGlob = flag.String("glob", "*", "Only links files that match the given glob")
	flag.Parse()

	if *rootSrc == "" || *rootDst == "" {
		flag.Usage()
		return
	}

	if *shouldReplaceOld {
		if err := db.RemoveDB(*rootDst); !os.IsNotExist(err) && err != nil {
			fmt.Printf("Error removing existing database: %v\n", err)
			os.Exit(1)
		}
	} else if _, err := os.Stat(*rootDst); err == nil {
		// If the destination directory already exists, we don't want to overwrite it.
		return
	}

	glob, err := glob.Compile(*srcGlob)
	if err != nil {
		fmt.Printf("Failed to compile given glob: %v\n", err)
	}

	fmt.Printf("Generating database at %s from source %s\n", *rootDst, *rootSrc)
	// Generate database
	if err := db.GenerateDB(*rootSrc, *rootDst, *glob); err != nil {
		fmt.Printf("Error generating database: %v\n", err)
		os.Exit(1)
	}
}

func Commands() []cli.Command {
	return nil
}

type MossContext struct {
	DataDir string
}
