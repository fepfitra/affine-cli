package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tomohiro-owada/affine-cli/internal/output"
)

func init() {
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Describe available commands and their parameters",
	Long:  "Output a machine-readable description of all commands, flags, and their types. Useful for AI agents to discover capabilities.",
	RunE: func(cmd *cobra.Command, args []string) error {
		commands := buildSchema(rootCmd)
		output.JSON(map[string]any{
			"version":  "1.0",
			"commands": commands,
		})
		return nil
	},
}

func buildSchema(cmd *cobra.Command) []map[string]any {
	var commands []map[string]any
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}

		if sub.HasSubCommands() {
			// Group command — recurse
			entry := map[string]any{
				"name":        sub.CommandPath(),
				"short":       sub.Short,
				"subcommands": buildSchema(sub),
			}
			commands = append(commands, entry)
		} else {
			// Leaf command
			entry := map[string]any{
				"name":  sub.CommandPath(),
				"short": sub.Short,
			}
			flags := extractFlags(sub)
			if len(flags) > 0 {
				entry["flags"] = flags
			}
			commands = append(commands, entry)
		}
	}
	return commands
}

func extractFlags(cmd *cobra.Command) []map[string]any {
	var flags []map[string]any
	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		entry := map[string]any{
			"name":     f.Name,
			"type":     f.Value.Type(),
			"default":  f.DefValue,
			"usage":    f.Usage,
			"required": isRequired(cmd, f.Name),
		}
		if f.Shorthand != "" {
			entry["shorthand"] = f.Shorthand
		}
		flags = append(flags, entry)
	})
	return flags
}

func isRequired(cmd *cobra.Command, flagName string) bool {
	ann := cmd.Flag(flagName)
	if ann == nil {
		return false
	}
	if ann.Annotations == nil {
		return false
	}
	_, ok := ann.Annotations[cobra.BashCompOneRequiredFlag]
	return ok
}
