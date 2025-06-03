package main

import (
	"github.com/Fordjour12/gcp-wif/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		cmd.HandleError(err)
	}
}
