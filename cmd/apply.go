package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/ghapi"
	"github.com/nobuo-miura/gh-new-repo/internal/repo"
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <owner/repo>",
		Short: "Apply a profile's settings to an existing repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runApply(args[0])
		},
	}
	cmd.Flags().StringVar(&flags.profile, "profile", "", "profile to apply (default: config default_profile)")
	return cmd
}

func runApply(target string) error {
	owner, name, ok := strings.Cut(target, "/")
	if !ok || owner == "" || name == "" {
		return errors.New("target must be in the form owner/repo")
	}

	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}
	_, profile, err := cfg.Resolve(flags.profile)
	if err != nil {
		if errors.Is(err, config.ErrProfileNotFound) {
			return fmt.Errorf(s.ProfileNotFound, firstNonEmpty(flags.profile, cfg.DefaultProfile))
		}
		return err
	}

	cl, err := ghapi.New()
	if err != nil {
		return err
	}

	plan := repo.FromProfile(profile)
	fmt.Fprintf(os.Stderr, "→ %s: %s/%s\n", s.StepApplyingConfig, owner, name)
	if err := repo.ApplyExisting(cl, owner, name, plan); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ %s\n", s.StepConfigApplied)

	if applyRulesets(s, owner, name, plan.Rulesets) {
		return errSilent
	}
	return nil
}
