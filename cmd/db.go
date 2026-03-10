package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbCreateCmd)
	dbCmd.AddCommand(dbAddColumnCmd)
	dbCmd.AddCommand(dbAddRowCmd)

	dbCreateCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = dbCreateCmd.MarkFlagRequired("doc-id")
	dbCreateCmd.Flags().String("title", "", "Database title")
	dbCreateCmd.Flags().StringSlice("columns", nil, "Column names (e.g., --columns Name,Status,Score)")

	dbAddColumnCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = dbAddColumnCmd.MarkFlagRequired("doc-id")
	dbAddColumnCmd.Flags().String("db-block-id", "", "Database block ID (required)")
	_ = dbAddColumnCmd.MarkFlagRequired("db-block-id")
	dbAddColumnCmd.Flags().String("name", "", "Column name (required)")
	_ = dbAddColumnCmd.MarkFlagRequired("name")
	dbAddColumnCmd.Flags().String("type", "rich-text", "Column type (rich-text, number, select, checkbox, date, etc.)")
	dbAddColumnCmd.Flags().Int("index", -1, "Insert position (-1 for end)")

	dbAddRowCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = dbAddRowCmd.MarkFlagRequired("doc-id")
	dbAddRowCmd.Flags().String("db-block-id", "", "Database block ID (required)")
	_ = dbAddRowCmd.MarkFlagRequired("db-block-id")
	dbAddRowCmd.Flags().String("cells", "{}", "Cell values as JSON: {\"col-id\": \"value\", ...}")
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage database blocks",
}

var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a database block in a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		title, _ := cmd.Flags().GetString("title")
		columns, _ := cmd.Flags().GetStringSlice("columns")
		if len(columns) == 1 && strings.Contains(columns[0], ",") {
			columns = strings.Split(columns[0], ",")
		}

		if isDryRun(cmd) {
			output.DryRun("create database", map[string]any{
				"doc_id": docID, "title": title, "columns": columns,
			})
			return nil
		}

		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()

		dbBlockID, colIDs, err := sess.CreateDatabase(docID, title, columns)
		if err != nil {
			return err
		}
		colMap := map[string]string{}
		if len(colIDs) > 0 {
			colMap["Title"] = colIDs[0]
		}
		for i, name := range columns {
			if i+1 < len(colIDs) {
				colMap[name] = colIDs[i+1]
			}
		}
		output.JSON(map[string]any{
			"created":     true,
			"db_block_id": dbBlockID,
			"doc_id":      docID,
			"columns":     colMap,
		})
		return nil
	},
}

var dbAddColumnCmd = &cobra.Command{
	Use:   "add-column",
	Short: "Add a column to a database block",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		dbBlockID, _ := cmd.Flags().GetString("db-block-id")
		if err := validate.SafeString("db-block-id", dbBlockID); err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		if err := validate.SafeString("name", name); err != nil {
			return err
		}
		colType, _ := cmd.Flags().GetString("type")
		index, _ := cmd.Flags().GetInt("index")

		if isDryRun(cmd) {
			output.DryRun("add database column", map[string]any{
				"doc_id": docID, "db_block_id": dbBlockID,
				"name": name, "type": colType, "index": index,
			})
			return nil
		}

		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()

		colID, err := sess.AddDatabaseColumn(docID, dbBlockID, name, colType, index)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{
			"added": true, "column_id": colID,
			"doc_id": docID, "db_block_id": dbBlockID,
		})
		return nil
	},
}

var dbAddRowCmd = &cobra.Command{
	Use:   "add-row",
	Short: "Add a row to a database block",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		dbBlockID, _ := cmd.Flags().GetString("db-block-id")
		if err := validate.SafeString("db-block-id", dbBlockID); err != nil {
			return err
		}
		cellsStr, _ := cmd.Flags().GetString("cells")
		cellsStr = strings.TrimSpace(cellsStr)
		if cellsStr == "" {
			cellsStr = "{}"
		}

		var cells map[string]string
		if err := json.Unmarshal([]byte(cellsStr), &cells); err != nil {
			return fmt.Errorf("invalid cells JSON: %w", err)
		}

		if isDryRun(cmd) {
			output.DryRun("add database row", map[string]any{
				"doc_id": docID, "db_block_id": dbBlockID, "cells": cells,
			})
			return nil
		}

		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()

		rowID, err := sess.AddDatabaseRow(docID, dbBlockID, cells)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{
			"added": true, "row_id": rowID,
			"doc_id": docID, "db_block_id": dbBlockID,
		})
		return nil
	},
}
