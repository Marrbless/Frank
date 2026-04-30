package main

import (
	"encoding/json"
	"fmt"

	"github.com/local/picobot/internal/missioncontrol"
	"github.com/spf13/cobra"
)

func newMissionStoreTransferPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "store-transfer-plan",
		Short:        "Print a dry-run mission store backup or restore plan",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			directionValue, _ := cmd.Flags().GetString("direction")
			if directionValue == "" {
				return fmt.Errorf("--direction is required")
			}

			storeRoot, _ := cmd.Flags().GetString("mission-store-root")
			if storeRoot == "" {
				return fmt.Errorf("--mission-store-root is required")
			}

			snapshotRoot, _ := cmd.Flags().GetString("snapshot-root")
			if snapshotRoot == "" {
				return fmt.Errorf("--snapshot-root is required")
			}

			plan, err := missioncontrol.BuildStoreTransferPlan(
				storeRoot,
				snapshotRoot,
				missioncontrol.StoreTransferPlanDirection(directionValue),
			)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode store transfer plan output: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write store transfer plan output: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().String("direction", "", "Transfer direction: backup or restore")
	cmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	cmd.Flags().String("snapshot-root", "", "Path to the backup snapshot root")
	return cmd
}
