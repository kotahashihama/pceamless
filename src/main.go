package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// バックアップする項目
type BackupItem struct {
	// 名前
	Name string
	// 選択状態
	Selected bool
}

// -e フラグで指定された除外する dotfile のリスト
var excludeFiles []string

// ルートコマンドの定義
var rootCmd = &cobra.Command{
	Use:   "pceamless",
	Short: "PC 作業環境移行支援ツール",
	Long:  `pceamless は macOS の作業環境を別のPCに移行するためのツールです。`,
}

// backup コマンド
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "作業環境のバックアップを作成",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkBrewInstallation(); err != nil {
			return fmt.Errorf("Homebrew がインストールされていません: %v", err)
		}

		items := []BackupItem{
			{Name: "Homebrew 管轄のアプリケーション群", Selected: true},
			{Name: "dotfile 群", Selected: true},
			{Name: "Documents", Selected: true},
			{Name: "Pictures", Selected: true},
			{Name: "Downloads", Selected: true},
			{Name: "Movies", Selected: true},
			{Name: "Music", Selected: true},
		}

		selected, err := selectBackupItems(items)
		if err != nil {
			return err
		}

		return performBackup(selected)
	},
}

// restore コマンド
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "バックアップから作業環境を復元",
	RunE: func(cmd *cobra.Command, args []string) error {
		return performRestore()
	},
}

func init() {
	// コマンドラインフラグとサブコマンドを設定
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)

	// -e フラグの設定
	backupCmd.Flags().StringSliceVarP(&excludeFiles, "exclude", "e", []string{}, "バックアップから除外する dotfile")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// brew コマンドがインストールされているかを確認
func checkBrewInstallation() error {
	_, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("Homebrew がインストールされていません。https://brew.sh/ からインストールしてください")
	}
	return nil
}

// バックアップ項目を選択する UI を提供
func selectBackupItems(items []BackupItem) ([]BackupItem, error) {
	options := make([]string, len(items))
	defaultSelected := make([]string, len(items))

	for i, item := range items {
		options[i] = item.Name
		if item.Selected {
			defaultSelected[i] = item.Name
		}
	}

	var selected []string
	prompt := &survey.MultiSelect{
		Message: "バックアップする項目を選択してください",
		Options: options,
		Default: defaultSelected,
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}

	selectedMap := make(map[string]bool)
	for _, s := range selected {
		selectedMap[s] = true
	}

	for i := range items {
		items[i].Selected = selectedMap[items[i].Name]
	}

	return items, nil
}
