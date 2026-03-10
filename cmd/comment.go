package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(commentCmd)
	commentCmd.AddCommand(commentListCmd)
	commentCmd.AddCommand(commentCreateCmd)
	commentCmd.AddCommand(commentUpdateCmd)
	commentCmd.AddCommand(commentDeleteCmd)
	commentCmd.AddCommand(commentResolveCmd)

	commentListCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = commentListCmd.MarkFlagRequired("doc-id")
	commentListCmd.Flags().Int("first", 20, "Number of comments")
	commentListCmd.Flags().Int("offset", 0, "Offset")
	commentListCmd.Flags().String("after", "", "Cursor")

	commentCreateCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = commentCreateCmd.MarkFlagRequired("doc-id")
	commentCreateCmd.Flags().String("content", "", "Comment content (required)")
	_ = commentCreateCmd.MarkFlagRequired("content")

	commentUpdateCmd.Flags().String("id", "", "Comment ID (required)")
	_ = commentUpdateCmd.MarkFlagRequired("id")
	commentUpdateCmd.Flags().String("content", "", "New content (required)")
	_ = commentUpdateCmd.MarkFlagRequired("content")

	commentDeleteCmd.Flags().String("id", "", "Comment ID (required)")
	_ = commentDeleteCmd.MarkFlagRequired("id")

	commentResolveCmd.Flags().String("id", "", "Comment ID (required)")
	_ = commentResolveCmd.MarkFlagRequired("id")
	commentResolveCmd.Flags().Bool("resolved", true, "Set resolved status")
}

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage comments",
}

var commentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments on a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		first, _ := cmd.Flags().GetInt("first")
		offset, _ := cmd.Flags().GetInt("offset")
		after, _ := cmd.Flags().GetString("after")
		vars := map[string]any{
			"workspaceId": ws,
			"docId":       docID,
			"first":       first,
			"offset":      offset,
		}
		if after != "" {
			vars["after"] = after
		}
		data, err := gql.Request(ctx(), graphql.ListCommentsQuery, vars)
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

var commentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		content, _ := cmd.Flags().GetString("content")
		if err := validate.SafeString("content", content); err != nil {
			return err
		}

		if isDryRun(cmd) {
			output.DryRun("create comment", map[string]any{
				"workspace_id": ws,
				"doc_id":       docID,
				"content":      content,
			})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.CreateCommentMutation, map[string]any{
			"input": map[string]any{
				"workspaceId": ws,
				"docId":       docID,
				"content":     map[string]any{"text": content},
			},
		})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var commentUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := validate.SafeString("id", id); err != nil {
			return err
		}
		content, _ := cmd.Flags().GetString("content")
		if err := validate.SafeString("content", content); err != nil {
			return err
		}

		if isDryRun(cmd) {
			output.DryRun("update comment", map[string]any{
				"comment_id": id,
				"content":    content,
			})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.UpdateCommentMutation, map[string]any{
			"input": map[string]any{
				"id":      id,
				"content": map[string]any{"text": content},
			},
		})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := validate.SafeString("id", id); err != nil {
			return err
		}

		if isDryRun(cmd) {
			output.DryRun("delete comment", map[string]any{
				"comment_id": id,
			})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.DeleteCommentMutation, map[string]any{"id": id})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var commentResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve or unresolve a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := validate.SafeString("id", id); err != nil {
			return err
		}
		resolved, _ := cmd.Flags().GetBool("resolved")

		if isDryRun(cmd) {
			output.DryRun("resolve comment", map[string]any{
				"comment_id": id,
				"resolved":   resolved,
			})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.ResolveCommentMutation, map[string]any{
			"input": map[string]any{
				"id":       id,
				"resolved": resolved,
			},
		})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}
