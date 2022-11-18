package main

import (
	"fmt"
	"github.com/DrmagicE/gmqtt/cmd/gmqctl/command"
	"github.com/spf13/cobra"
	"os"
)

var (
	rootCmd = &cobra.Command{
		Use:     "gmqctl",
		Long:    "gmqctl is a command line tool for gmqtt",
		Version: Version,
	}
)

func init() {
	rootCmd.AddCommand(command.Gen)
}

func must(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	must(rootCmd.Execute())
}
