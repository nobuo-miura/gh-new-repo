package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/nobuo-miura/gh-new-repo/internal/i18n"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Manage extension settings",
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "language [auto|en|ja]",
			Short: "Show or set the UI language",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				if len(args) == 1 {
					return runConfigSetLanguage(args[0])
				}
				return runConfigPickLanguage()
			},
		},
		&cobra.Command{
			Use:   "show",
			Short: "Print extension settings",
			Args:  cobra.NoArgs,
			RunE:  func(_ *cobra.Command, _ []string) error { return runConfigShow() },
		},
	)
	return c
}

func normalizeLanguage(v string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "auto", "":
		return "auto", true
	case "en", "english":
		return "en", true
	case "ja", "japanese", "日本語":
		return "ja", true
	default:
		return "", false
	}
}

func saveLanguage(value string) error {
	cfg, _, err := loadForLang()
	if err != nil {
		return err
	}
	cfg.Language = value
	if err := cfg.Save(); err != nil {
		return err
	}
	// 設定後の言語で完了メッセージを表示します。
	s := i18n.For(i18n.Detect(firstNonEmpty(flags.lang, value)))
	fmt.Fprintf(os.Stderr, s.LanguageSet+"\n", value)
	return nil
}

func runConfigSetLanguage(arg string) error {
	value, ok := normalizeLanguage(arg)
	if !ok {
		_, s, err := loadForLang()
		if err != nil {
			return err
		}
		return errors.New(s.InvalidLanguage)
	}
	return saveLanguage(value)
}

func runConfigPickLanguage() error {
	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}

	selected := firstNonEmpty(cfg.Language, "auto")
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title(s.LanguageSettingTitle).Description(s.LanguageSettingHelp).
			Options(
				huh.NewOption(s.LangAuto, "auto"),
				huh.NewOption(s.LangEnglish, "en"),
				huh.NewOption(s.LangJapanese, "ja"),
			).Value(&selected),
	))
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Fprintln(os.Stderr, s.Cancelled)
			return nil
		}
		return err
	}
	return saveLanguage(selected)
}

func runConfigShow() error {
	cfg, _, err := loadForLang()
	if err != nil {
		return err
	}
	fmt.Printf("language:        %s\n", firstNonEmpty(cfg.Language, "auto"))
	fmt.Printf("default_profile: %s\n", cfg.DefaultProfile)
	fmt.Printf("config:          %s\n", cfg.FilePath())
	return nil
}
