package cmd

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(blobCmd)
	blobCmd.AddCommand(blobUploadCmd)
	blobCmd.AddCommand(blobDeleteCmd)
	blobCmd.AddCommand(blobCleanupCmd)

	blobUploadCmd.Flags().String("file", "", "File path to upload (required)")
	_ = blobUploadCmd.MarkFlagRequired("file")

	blobDeleteCmd.Flags().String("key", "", "Blob key (required)")
	_ = blobDeleteCmd.MarkFlagRequired("key")
	blobDeleteCmd.Flags().Bool("permanently", false, "Delete permanently")
}

var blobCmd = &cobra.Command{
	Use:   "blob",
	Short: "Manage blob storage",
}

var blobUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file as a blob",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		filePath, _ := cmd.Flags().GetString("file")
		if err := validate.SafeString("file", filePath); err != nil {
			return err
		}

		if isDryRun(cmd) {
			output.DryRun("upload blob", map[string]any{
				"workspace_id": ws, "file": filePath,
			})
			return nil
		}

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		// Build multipart request
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Operations part
		ops := fmt.Sprintf(`{"query":%q,"variables":{"workspaceId":%q,"blob":null}}`,
			graphql.SetBlobMutation, ws)
		_ = writer.WriteField("operations", ops)
		_ = writer.WriteField("map", `{"0":["variables.blob"]}`)

		// File part
		part, err := writer.CreateFormFile("0", filepath.Base(filePath))
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, f); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
		writer.Close()

		data, err := gql.RequestMultipart(ctx(), &buf, writer.FormDataContentType())
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var blobDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a blob",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		key, _ := cmd.Flags().GetString("key")
		if err := validate.SafeString("key", key); err != nil {
			return err
		}
		perm, _ := cmd.Flags().GetBool("permanently")
		if isDryRun(cmd) {
			output.DryRun("delete blob", map[string]any{
				"workspace_id": ws, "key": key, "permanently": perm,
			})
			return nil
		}
		data, err := gql.Request(ctx(), graphql.DeleteBlobMutation, map[string]any{
			"workspaceId": ws, "key": key, "permanently": perm,
		})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var blobCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Permanently release deleted blobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := requireWorkspace()
		if err != nil {
			return err
		}
		if isDryRun(cmd) {
			output.DryRun("cleanup blobs", map[string]any{"workspace_id": ws})
			return nil
		}
		data, err := gql.Request(ctx(), graphql.CleanupBlobsMutation, map[string]any{
			"workspaceId": ws,
		})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}
