package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysNewCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create"},
	Short:   "Create a new key",
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
		existingKeyNames := make([]string, len(existingKeys))
		for i, key := range existingKeys {
			existingKeyNames[i] = key.Name
		}

		var name string
		readOnly := true

		nameInput := huh.NewInput().
			Title("Key name ").
			Placeholder("something_you_will_recognize_later").
			Value(&name).
			CharLimit(64).
			Inline(true).
			Validate(func(s string) error {
				s = strings.TrimSpace(s)
				if s == "" {
					return fmt.Errorf("key name cannot be empty")
				}
				if slices.Contains(existingKeyNames, s) {
					return fmt.Errorf("key name must be unique, %q already exists", s)
				}
				return nil
			})
		readOnlyInput := huh.NewConfirm().
			Title("Is this key read-only?").
			Value(&readOnly).
			Inline(true)

		err = nameInput.Run()
		if err != nil {
			return err
		}
		err = readOnlyInput.Run()
		if err != nil {
			return err
		}

		name = strings.TrimSpace(name)

		key, err := adminClient.CreateKey(name, readOnly)
		if err != nil {
			return fmt.Errorf("cannot create key: %w", err)
		}

		fmt.Printf("Key created, you will only see it once so save it now:\n\n%s\n", *key.Key)

		return nil
	},
}

func init() {
	keyCmd.AddCommand(keysNewCmd)
}
