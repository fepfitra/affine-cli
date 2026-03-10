package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyListCmd)

	historyListCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = historyListCmd.MarkFlagRequired("doc-id")
	historyListCmd.Flags().Int("take", 10, "Number of history entries")
	historyListCmd.Flags().String("before", "", "Before timestamp (ISO 8601)")
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage document history",
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List document version history",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		take, _ := cmd.Flags().GetInt("take")
		vars := map[string]any{
			"workspaceId": ws,
			"guid":        docID,
			"take":        take,
		}
		if before, _ := cmd.Flags().GetString("before"); before != "" {
			vars["before"] = before
		}
		data, err := gql.Request(ctx(), graphql.ListHistoriesQuery, vars)
		if err != nil {
			return err
		}
		if fields := getFields(cmd); len(fields) > 0 {
			output.FilteredJSON(data, fields)
			return nil
		}
		output.RawJSON(data)
		return nil
	},
}
