package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagListDocsCmd)

	tagCreateCmd.Flags().String("name", "", "Tag name (required)")
	_ = tagCreateCmd.MarkFlagRequired("name")

	tagAddCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = tagAddCmd.MarkFlagRequired("doc-id")
	tagAddCmd.Flags().String("tag", "", "Tag name (required)")
	_ = tagAddCmd.MarkFlagRequired("tag")

	tagRemoveCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = tagRemoveCmd.MarkFlagRequired("doc-id")
	tagRemoveCmd.Flags().String("tag", "", "Tag name (required)")
	_ = tagRemoveCmd.MarkFlagRequired("tag")

	tagListDocsCmd.Flags().String("tag", "", "Tag name (required)")
	_ = tagListDocsCmd.MarkFlagRequired("tag")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()
		tags, err := sess.ListTags()
		if err != nil {
			return err
		}
		output.JSON(map[string]any{
			"tags":  tags,
			"count": len(tags),
		})
		return nil
	},
}

var tagCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if err := validate.SafeString("name", name); err != nil {
			return err
		}
		if isDryRun(cmd) {
			output.DryRun("create tag", map[string]any{"name": name})
			return nil
		}
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()
		tagID, err := sess.CreateTag(name)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{"created": true, "tag_id": tagID, "name": name})
		return nil
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a tag to a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		tag, _ := cmd.Flags().GetString("tag")
		if err := validate.SafeString("tag", tag); err != nil {
			return err
		}
		if isDryRun(cmd) {
			output.DryRun("add tag to doc", map[string]any{"doc_id": docID, "tag": tag})
			return nil
		}
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()
		err = sess.AddTagToDoc(docID, tag)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{"added": true, "doc_id": docID, "tag": tag})
		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a tag from a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		tag, _ := cmd.Flags().GetString("tag")
		if err := validate.SafeString("tag", tag); err != nil {
			return err
		}
		if isDryRun(cmd) {
			output.DryRun("remove tag from doc", map[string]any{"doc_id": docID, "tag": tag})
			return nil
		}
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()
		err = sess.RemoveTagFromDoc(docID, tag)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{"removed": true, "doc_id": docID, "tag": tag})
		return nil
	},
}

var tagListDocsCmd = &cobra.Command{
	Use:   "list-docs",
	Short: "List documents with a specific tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		if err := validate.SafeString("tag", tag); err != nil {
			return err
		}
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()
		docIDs, err := sess.ListDocsByTag(tag)
		if err != nil {
			return err
		}
		output.JSON(map[string]any{
			"tag":     tag,
			"doc_ids": docIDs,
			"count":   len(docIDs),
		})
		return nil
	},
}
