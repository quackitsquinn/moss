package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/quackitsquinn/moss/internal/db"
)

func main() {
	var rootSrc = flag.String("root_src", "", "Root source directory, supports globbing")
	var rootDst = flag.String("root_dst", "", "Root destination directory")
	var shouldReplaceOld = flag.Bool("replace", false, "Replace existing database")
	var ext = flag.String("ext", ".png", "File extension to include in the database")
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

	fmt.Printf("Generating database at %s from source %s\n", *rootDst, *rootSrc)
	// Generate database
	if err := db.GenerateDB(*rootSrc, *rootDst, *ext); err != nil {
		fmt.Printf("Error generating database: %v\n", err)
		os.Exit(1)
	}

}
