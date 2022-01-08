package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"

	"git.kausm.in/kaustubh/autosaved/core"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("error while opening repo: %v\n", err)
	}

	asd := core.Autosaved{
		Repository: repo,
		MinChars:   2000,
		MinMinutes: 2,
	}

	if len(os.Args) > 1 && os.Args[1] == "save" {
		err := asd.Save("manually committed")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}
