package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const maxKeyNameLength = 64

var (
	keysNewName     string
	keysNewReadOnly bool
	keysNewQuiet    bool
)

var keysNewCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create"},
	Short:   "Create a new key",
	Example: `
# Create new key interactively
artf key new

# Create new readonly key
artf key new --name mykey --readonly

# Create new writable key
artf key new --name mykey --readonly=false
`,
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

		validateName := func(s string) error {
			s = strings.TrimSpace(s)
			if s == "" {
				return fmt.Errorf("key name cannot be empty")
			}
			if slices.Contains(existingKeyNames, s) {
				return fmt.Errorf("key name must be unique, %q already exists", s)
			}
			if len(s) > maxKeyNameLength {
				return fmt.Errorf("key name cannot be longer than %d characters", maxKeyNameLength)
			}
			return nil
		}

		var name string
		readOnly := true

		if cmd.Flags().Changed("name") {
			name = strings.TrimSpace(keysNewName)
			if err := validateName(name); err != nil {
				return err
			}
		} else {
			nameInput := huh.NewInput().
				Title("Key name ").
				Placeholder("something_you_will_recognize_later").
				Value(&name).
				CharLimit(maxKeyNameLength).
				Inline(true).
				Validate(validateName)
			err = nameInput.Run()
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
		}

		if cmd.Flags().Changed("readonly") {
			readOnly = keysNewReadOnly
		} else {
			readOnlyInput := huh.NewConfirm().
				Title("Is this key read-only?").
				Value(&readOnly).
				Inline(true)
			err = readOnlyInput.Run()
			if err != nil {
				return err
			}
		}

		key, err := adminClient.CreateKey(name, readOnly)
		if err != nil {
			return fmt.Errorf("cannot create key: %w", err)
		}

		if keysNewQuiet {
			fmt.Println(*key.Key)
		} else {
			fmt.Printf("Key created, you will only see it once so save it now:\n\n%s\n", *key.Key)
		}

		return nil
	},
}

func init() {
	keyCmd.AddCommand(keysNewCmd)
	keysNewCmd.Flags().StringVarP(&keysNewName, "name", "n", "", "key name")
	keysNewCmd.Flags().BoolVar(&keysNewReadOnly, "readonly", true, "read-only key")
	keysNewCmd.Flags().BoolVarP(&keysNewQuiet, "quiet", "q", false, "print only key after creation")
}
