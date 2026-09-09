// Command price-checker estimates the AWS cost of a Terraform plan.
package main

import (
	"os"

	"github.com/osifotos/price-checker/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
