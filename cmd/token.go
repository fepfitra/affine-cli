package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
	"github.com/tomohiro-owada/affine-cli/internal/output"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenGenerateCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)

	tokenGenerateCmd.Flags().String("name", "", "Token name (required)")
	_ = tokenGenerateCmd.MarkFlagRequired("name")
	tokenGenerateCmd.Flags().String("expires-at", "", "Expiration date (ISO 8601)")

	tokenRevokeCmd.Flags().String("id", "", "Token ID (required)")
	_ = tokenRevokeCmd.MarkFlagRequired("id")
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage access tokens",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List access tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := gql.Request(ctx(), graphql.ListAccessTokensQuery, nil)
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

var tokenGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if err := validate.SafeString("name", name); err != nil {
			return err
		}

		if isDryRun(cmd) {
			details := map[string]any{"name": name}
			if expiresAt, _ := cmd.Flags().GetString("expires-at"); expiresAt != "" {
				details["expires_at"] = expiresAt
			}
			output.DryRun("generate access token", details)
			return nil
		}

		input := map[string]any{"name": name}
		if expiresAt, _ := cmd.Flags().GetString("expires-at"); expiresAt != "" {
			input["expiresAt"] = expiresAt
		}
		data, err := gql.Request(ctx(), graphql.GenerateAccessTokenMutation, map[string]any{"input": input})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := validate.SafeString("id", id); err != nil {
			return err
		}

		if isDryRun(cmd) {
			output.DryRun("revoke access token", map[string]any{
				"token_id": id,
			})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.RevokeAccessTokenMutation, map[string]any{"id": id})
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}
