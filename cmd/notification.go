package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
	"github.com/tomohiro-owada/affine-cli/internal/output"
)

func init() {
	rootCmd.AddCommand(notificationCmd)
	notificationCmd.AddCommand(notificationListCmd)
	notificationCmd.AddCommand(notificationReadAllCmd)

	notificationListCmd.Flags().Int("first", 20, "Number of notifications")
	notificationListCmd.Flags().Int("offset", 0, "Offset")
	notificationListCmd.Flags().String("after", "", "Cursor")
}

var notificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Manage notifications",
}

var notificationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notifications",
	RunE: func(cmd *cobra.Command, args []string) error {
		first, _ := cmd.Flags().GetInt("first")
		offset, _ := cmd.Flags().GetInt("offset")
		after, _ := cmd.Flags().GetString("after")
		pagination := map[string]any{
			"first":  first,
			"offset": offset,
		}
		if after != "" {
			pagination["after"] = after
		}
		data, err := gql.Request(ctx(), graphql.ListNotificationsQuery, map[string]any{
			"pagination": pagination,
		})
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

var notificationReadAllCmd = &cobra.Command{
	Use:   "read-all",
	Short: "Mark all notifications as read",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isDryRun(cmd) {
			output.DryRun("mark all notifications as read", map[string]any{})
			return nil
		}

		data, err := gql.Request(ctx(), graphql.ReadAllNotificationsMutation, nil)
		if err != nil {
			return err
		}
		output.RawJSON(data)
		return nil
	},
}
