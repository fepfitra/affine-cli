package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(tableCmd)
	tableCmd.AddCommand(tableAddRowCmd)

	tableAddRowCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = tableAddRowCmd.MarkFlagRequired("doc-id")
	tableAddRowCmd.Flags().String("block-id", "", "Table block ID (required)")
	_ = tableAddRowCmd.MarkFlagRequired("block-id")
	tableAddRowCmd.Flags().StringSlice("cells", nil, "Cell values (e.g., --cells 'val1,val2,val3')")
}

var tableCmd = &cobra.Command{
	Use:   "table",
	Short: "Manage table blocks",
}

var tableAddRowCmd = &cobra.Command{
	Use:   "add-row",
	Short: "Add a row to a table block",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		blockID, _ := cmd.Flags().GetString("block-id")
		if err := validate.SafeString("block-id", blockID); err != nil {
			return err
		}
		cells, _ := cmd.Flags().GetStringSlice("cells")
		if len(cells) == 1 && strings.Contains(cells[0], ",") {
			cells = strings.Split(cells[0], ",")
		}

		if isDryRun(cmd) {
			output.DryRun("add table row", map[string]any{
				"doc_id": docID, "block_id": blockID, "cells": cells,
			})
			return nil
		}

		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()

		rowID, err := sess.AddTableRow(docID, blockID, cells)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{
			"added": true, "row_id": rowID,
			"doc_id": docID, "block_id": blockID,
		})
		return nil
	},
}
