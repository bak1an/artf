package cmd

import (
	"fmt"
	"strconv"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysRmYes bool

var keysRmCmd = &cobra.Command{
	Use:     "rm [id]",
	Aliases: []string{"delete"},
	Short:   "Delete a key",
	Example: `
# Delete key interactively
artf key rm

# Delete key by id
artf key rm 1

# Delete key by id skipping confirmation
artf key rm 1 --yes
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")

		cmd.SilenceUsage = true

		adminClient, err := initAdminClient(data)
		if err != nil {
			return err
		}

		existingKeys, err := adminClient.ListKeys()
		if err != nil {
			return err
		}

		if len(existingKeys) == 0 {
			fmt.Println("No keys exist so far. You might want to create one with `artf keys create`")
			return nil
		}

		var toDelete *admin.Key

		if len(args) == 1 {
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid key id %q: %w", args[0], err)
			}
			for _, k := range existingKeys {
				if k.ID == id {
					toDelete = k
					break
				}
			}
			if toDelete == nil {
				return fmt.Errorf("key with id %d not found", id)
			}
		} else {
			options := make([]huh.Option[*admin.Key], len(existingKeys))
			for i, key := range existingKeys {
				options[i] = huh.Option[*admin.Key]{
					Key:   key.Name,
					Value: key,
				}
			}
			toDeleteInput := huh.NewSelect[*admin.Key]().
				Title("Select key to delete (type / to filter)").
				Options(options...).
				Value(&toDelete)

			err = toDeleteInput.Run()
			if err != nil {
				return err
			}
		}

		confirm := keysRmYes
		if !confirm {
			confirmInput := huh.NewConfirm().
				Title(fmt.Sprintf("Are you sure you want to delete key '%s'?", toDelete.Name)).
				Value(&confirm).
				Inline(true)
			err = confirmInput.Run()
			if err != nil {
				return err
			}
		}

		if !confirm {
			fmt.Println("Ok, doing nothing")
			return nil
		}

		err = adminClient.DeleteKey(toDelete.ID)
		if err != nil {
			return fmt.Errorf("cannot delete key: %w", err)
		}
		fmt.Println("Key deleted")

		return nil
	},
}

func init() {
	keyCmd.AddCommand(keysRmCmd)
	keysRmCmd.Flags().BoolVar(&keysRmYes, "yes", false, "skip confirmation")
}
