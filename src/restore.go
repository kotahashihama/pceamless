package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// リストアプロセス全体を制御
func performRestore() error {
	desktop := os.ExpandEnv("$HOME/Desktop")
	backupZip, err := findLatestBackupZip(desktop)
	if err != nil {
		return err
	}

	backupDir := strings.TrimSuffix(backupZip, ".zip")
	if err := unzip(backupZip, backupDir); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(backupDir, "Brewfile")); err == nil {
		fmt.Println("Homebrew 管轄のアプリケーション群をリストア中...")
		if err := restoreHomebrew(backupDir); err != nil {
			return fmt.Errorf("Homebrew 管轄のアプリケーション群のリストアに失敗しました: %v", err)
		}
	}

	if _, err := os.Stat(filepath.Join(backupDir, "dotfiles")); err == nil {
		fmt.Println("dotfile 群をリストア中...")
		if err := restoreDotfiles(backupDir); err != nil {
			return fmt.Errorf("dotfile 群のリストアに失敗しました: %v", err)
		}
	}

	if _, err := os.Stat(filepath.Join(backupDir, "userfiles")); err == nil {
		fmt.Println("ユーザーファイル群をリストア中...")
		if err := restoreUserFiles(backupDir); err != nil {
			return fmt.Errorf("ユーザーファイル群のリストアに失敗しました: %v", err)
		}
	}

	// クリーンアップ
	if err := os.RemoveAll(backupDir); err != nil {
		fmt.Printf("WARNING: 一時ディレクトリの削除に失敗しました: %v\n", err)
	}
	if err := os.Remove(backupZip); err != nil {
		fmt.Printf("WARNING: バックアップ ZIP ファイルの削除に失敗しました: %v\n", err)
	}

	fmt.Println("リストア完了")
	return nil
}

// 最新のバックアップ ZIP ファイルを検索
func findLatestBackupZip(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var latest string
	var latestTime int64

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "pceamless_backup_") || !strings.HasSuffix(name, ".zip") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > latestTime {
			latest = filepath.Join(dir, name)
			latestTime = info.ModTime().Unix()
		}
	}

	if latest == "" {
		return "", fmt.Errorf("バックアップ ZIP ファイルが見つかりません")
	}

	return latest, nil
}

// Homebrew 管轄のアプリケーション群をリストア
func restoreHomebrew(backupDir string) error {
	cmd := exec.Command("brew", "bundle", "--file", filepath.Join(backupDir, "Brewfile"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// dotfile 群をリストア
func restoreDotfiles(backupDir string) error {
	dotfilesDir := filepath.Join(backupDir, "dotfiles")
	entries, err := os.ReadDir(dotfilesDir)
	if err != nil {
		return err
	}

	excludeMap := make(map[string]bool)
	for _, file := range restoreExcludeFiles {
		name := strings.TrimPrefix(file, ".")
		excludeMap[name] = true
		excludeMap["."+name] = true
	}

	homeDir := os.ExpandEnv("$HOME")
	for _, entry := range entries {
		name := entry.Name()

		if excludeMap[name] {
			fmt.Printf("INFO: %s はリストアから除外されました\n", name)
			continue
		}

		srcPath := filepath.Join(dotfilesDir, name)
		dstPath := filepath.Join(homeDir, name)

		if err := os.RemoveAll(dstPath); err != nil {
			fmt.Printf("WARNING: %s の削除に失敗しました: %v\n", dstPath, err)
		}

		if err := copyPath(srcPath, dstPath); err != nil {
			if os.IsPermission(err) {
				fmt.Printf("WARNING: アクセス権限の問題により %s のリストアに失敗しました: %v\n", name, err)
			} else {
				fmt.Printf("WARNING: %s のリストアに失敗しました: %v\n", name, err)
			}
			continue
		}
	}

	return nil
}

// ユーザーファイル群をリストア
func restoreUserFiles(backupDir string) error {
	userfilesDir := filepath.Join(backupDir, "userfiles")
	entries, err := os.ReadDir(userfilesDir)
	if err != nil {
		return err
	}

	homeDir := os.ExpandEnv("$HOME")
	for _, entry := range entries {
		name := entry.Name()
		srcPath := filepath.Join(userfilesDir, name)
		dstPath := filepath.Join(homeDir, name)

		if err := copyPath(srcPath, dstPath); err != nil {
			if os.IsPermission(err) {
				fmt.Printf("WARNING: アクセス権限の問題により %s のリストアに失敗しました: %v\n", name, err)
			} else {
				fmt.Printf("WARNING: %s のリストアに失敗しました: %v\n", name, err)
			}
			continue
		}
	}

	return nil
}

// ZIP ファイルを解凍
func unzip(src, dst string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(dst, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
