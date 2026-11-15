// Command zk runs a Zettelkasten slip box in an Obsidian vault.
package main

import (
	"os"

	"github.com/alexsasharegan/zk/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
