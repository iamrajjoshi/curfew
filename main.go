package main

import (
	"fmt"
	"os"

	"github.com/iamrajjoshi/curfew/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if coder, ok := err.(interface{ ExitCode() int }); ok {
			if err.Error() != "" {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(coder.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
