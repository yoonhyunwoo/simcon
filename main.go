/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/yoonhyunwoo/simcon/cmd"
	"github.com/yoonhyunwoo/simcon/pkg/config"

	"log"
	"os"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		log.Fatalf("cannot create data directory %s: %v", config.DataDir, err)
	}

	cmd.Execute()
}
