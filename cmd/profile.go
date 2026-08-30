package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/ghapi"
	"github.com/nobuo-miura/gh-new-repo/internal/tui"
)

func newProfileCmd() *cobra.Command {
	profile := &cobra.Command{
		Use:   "profile",
		Short: "Manage initial-value profiles",
	}

	var from string
	var setDefault bool
	newCmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a profile via an interactive form",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runProfileNew(args[0], from, setDefault)
		},
	}
	newCmd.Flags().StringVar(&from, "from", "", "seed values from an existing profile")
	newCmd.Flags().BoolVar(&setDefault, "set-default", false, "make the new profile the default")

	var editFile bool
	editCmd := &cobra.Command{
		Use:   "edit [<name>]",
		Short: "Edit a profile in a form (default profile if omitted; --file for the raw config)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runProfileEdit(name, editFile)
		},
	}
	editCmd.Flags().BoolVar(&editFile, "file", false, "open the raw config file in $EDITOR")

	profile.AddCommand(
		newCmd,
		editCmd,
		&cobra.Command{
			Use:   "list",
			Short: "List profiles",
			Args:  cobra.NoArgs,
			RunE:  func(_ *cobra.Command, _ []string) error { return runProfileList() },
		},
		&cobra.Command{
			Use:   "show <name>",
			Short: "Print a profile as YAML",
			Args:  cobra.ExactArgs(1),
			RunE:  func(_ *cobra.Command, args []string) error { return runProfileShow(args[0]) },
		},
		&cobra.Command{
			Use:   "default <name>",
			Short: "Set the default profile",
			Args:  cobra.ExactArgs(1),
			RunE:  func(_ *cobra.Command, args []string) error { return runProfileDefault(args[0]) },
		},
		&cobra.Command{
			Use:   "path",
			Short: "Print the config file path",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				p, err := config.Path()
				if err != nil {
					return err
				}
				fmt.Println(p)
				return nil
			},
		},
	)
	return profile
}

func runProfileList() error {
	cfg, _, err := loadForLang()
	if err != nil {
		return err
	}
	for _, name := range cfg.Names() {
		marker := "  "
		if name == cfg.DefaultProfile {
			marker = "* "
		}
		fmt.Printf("%s%s\n", marker, name)
	}
	return nil
}

func runProfileShow(name string) error {
	cfg, _, err := loadForLang()
	if err != nil {
		return err
	}
	_, profile, err := cfg.Resolve(name)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}

func runProfileDefault(name string) error {
	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf(s.ProfileNotFound, name)
	}
	cfg.DefaultProfile = name
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, s.DefaultProfileSet+"\n", name)
	return nil
}

func runProfileNew(name, from string, setDefault bool) error {
	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf(s.ProfileExists, name)
	}

	base := config.Profile{}
	switch {
	case from != "":
		src, ok := cfg.Profiles[from]
		if !ok {
			return fmt.Errorf(s.ProfileFromNotFound, from)
		}
		base = src
	default:
		if def, ok := cfg.Profiles[cfg.DefaultProfile]; ok {
			base = def
		}
	}

	cl, err := ghapi.New()
	if err != nil {
		return err
	}
	meta, err := loadMeta(cl)
	if err != nil {
		return err
	}

	saved, err := tui.RunProfileEdit(s, &base, meta)
	if err != nil {
		return err
	}
	if !saved {
		fmt.Fprintln(os.Stderr, s.Cancelled)
		return nil
	}

	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	cfg.Profiles[name] = base
	if setDefault || cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, s.ProfileCreated+"\n", name)
	return nil
}

func runProfileEdit(name string, rawFile bool) error {
	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}

	if rawFile {
		editor := firstNonEmpty(os.Getenv("GH_NEW_REPO_EDITOR"), os.Getenv("VISUAL"), os.Getenv("EDITOR"))
		if editor == "" {
			return errors.New(s.NoEditorEnv)
		}
		c := exec.Command(editor, cfg.FilePath()) //nolint:gosec // ユーザーが指定したエディターを実行します。
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		return c.Run()
	}

	if len(cfg.Profiles) == 0 {
		return errors.New(s.NoProfilesHint)
	}
	pname, profile, err := cfg.Resolve(name)
	if err != nil {
		return fmt.Errorf(s.ProfileNotFound, firstNonEmpty(name, cfg.DefaultProfile))
	}

	cl, err := ghapi.New()
	if err != nil {
		return err
	}
	meta, err := loadMeta(cl)
	if err != nil {
		return err
	}

	saved, err := tui.RunProfileEdit(s, &profile, meta)
	if err != nil {
		return err
	}
	if !saved {
		fmt.Fprintln(os.Stderr, s.Cancelled)
		return nil
	}

	cfg.Profiles[pname] = profile
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, s.ProfileUpdated+"\n", pname)
	return nil
}
