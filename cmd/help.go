package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// buildHelpMarkdown converts Cobra command metadata into formatted markdown.
func buildHelpMarkdown(cmd *cobra.Command) string {
	var b strings.Builder

	b.WriteString("# MOCHI v" + Version + "\n\n")

	if cmd.Long != "" {
		b.WriteString(cmd.Long + "\n\n")
	} else if cmd.Short != "" {
		b.WriteString(cmd.Short + "\n\n")
	}

	b.WriteString("## Usage\n\n")
	b.WriteString("```\n" + cmd.UseLine() + "\n```\n\n")

	if cmd.Example != "" {
		b.WriteString("## Examples\n\n")
		b.WriteString("```bash\n" + cmd.Example + "\n```\n\n")
	}

	if cmds := cmd.Commands(); len(cmds) > 0 {
		b.WriteString("## Commands\n\n")
		for _, c := range cmds {
			if c.Hidden {
				continue
			}
			aliases := ""
			if len(c.Aliases) > 0 {
				aliases = fmt.Sprintf(" (%s)", strings.Join(c.Aliases, ", "))
			}
			b.WriteString(fmt.Sprintf("- **%s**%s — %s\n", c.Name(), aliases, c.Short))
		}
		b.WriteString("\n")
	}

	flags := cmd.Flags()
	if flags.HasFlags() {
		b.WriteString("## Flags\n\n")
		b.WriteString("| Flag | Default | Description |\n")
		b.WriteString("|------|---------|-------------|\n")
		flags.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			name := "--" + f.Name
			if f.Shorthand != "" {
				name = "-" + f.Shorthand + ", " + name
			}
			def := f.DefValue
			if def == "" {
				def = "-"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", name, def, f.Usage))
		})
		b.WriteString("\n")
	}

	return b.String()
}
