package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:    "gen-rst [<directory to save files>]",
	Short:  "Generate RST docs for this tool",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	Run:    doGenDocs,
}

func doGenDocs(cmd *cobra.Command, args []string) {
	outDir := "./"
	if len(args) == 1 {
		outDir = args[0]
	}
	fmt.Println("Generating docs at:", outDir)

	filePrepender := func(filename string) string {
		return ":orphan:\n\n"
	}

	linkHandler := func(name, ref string) string {
		return fmt.Sprintf(":ref:`%s <%s>`", name, ref)
	}

	rootCmd.DisableAutoGenTag = true
	err := doc.GenReSTTreeCustom(rootCmd, outDir, filePrepender, linkHandler)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}
