package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/TuneLab/gob/truss/data"
	"path/filepath"
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: truss microservice.proto\n")
		os.Exit(1)
	}

	definitionPath := flag.Arg(0)
	fmt.Println(definitionPath)

	wd, _ := os.Getwd()
	fmt.Println(wd)

	for _, filePath := range data.AssetNames() {
		fileBytes, err := data.Asset(filePath)
		check(err)
		fullPath := wd + "/" + filePath

		fullPathDir := filepath.Dir(fullPath)
		os.MkdirAll(fullPathDir, 0777)

		err = ioutil.WriteFile(fullPath, fileBytes, 0666)
		check(err)
	}
}
func check(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

}
