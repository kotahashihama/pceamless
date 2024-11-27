package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// バックアッププロセス全体を制御
func performBackup(items []BackupItem) error {
	timestamp := time.Now().Format("20060102150405") // YYYYMMDDHHmmss 形式
	backupDir := filepath.Join(os.ExpandEnv("$HOME/Desktop"), fmt.Sprintf("pceamless_backup_%s", timestamp))

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("バックアップディレクトリの作成に失敗しました: %v", err)
	}

	for _, item := range items {
		if !item.Selected {
			continue
		}

		switch item.Name {
		case "Homebrew 管轄のアプリケーション群":
			fmt.Println("Homebrew 管轄のアプリケーション群をバックアップ中...")
			if err := backupHomebrew(backupDir); err != nil {
				return fmt.Errorf("Homebrew 管轄のアプリケーション群のバックアップに失敗しました: %v", err)
			}

		case "dotfile群":
			fmt.Println("dotfile 群をバックアップ中...")
			if err := backupDotfiles(backupDir); err != nil {
				return fmt.Errorf("dotfile のバックアップに失敗しました: %v", err)
			}

		default:
			// Documents, Pictures などのユーザーディレクトリの処理
			fmt.Printf("%s をバックアップ中...\n", item.Name)
			if err := backupUserFiles(backupDir, item.Name); err != nil {
				return fmt.Errorf("%s のバックアップに失敗しました: %v", item.Name, err)
			}
		}
	}

	zipFile := backupDir + ".zip"
	if err := createZip(backupDir, zipFile); err != nil {
		return fmt.Errorf("ZIP ファイルの作成に失敗しました: %v", err)
	}

	// クリーンアップ
	if err := os.RemoveAll(backupDir); err != nil {
		fmt.Printf("WARNING: バックアップ用の一時ディレクトリの削除に失敗しました: %v\n", err)
	}

	fmt.Printf("バックアップ完了。デスクトップに作られた ZIP ファイルを新しい PC のデスクトップにコピーしてください\n")
	return nil
}

// バックアップしたファイルを ZIP ファイルで圧縮
func createZip(src, dst string) error {
	zipfile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("ZIP ファイルの作成に失敗しました: %v", err)
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("WARNING: %sへのアクセス中にエラーが発生しました: %v\n", path, err)
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("INFO: シンボリックリンク %s をスキップします\n", path)
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			fmt.Printf("WARNING: %s の相対パス計算に失敗しました: %v\n", path, err)
			return nil
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			fmt.Printf("WARNING: %s のファイル情報取得に失敗しました: %v\n", path, err)
			return nil
		}
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
			_, err = archive.CreateHeader(header)
			if err != nil {
				fmt.Printf("WARNING: ディレクトリ %s の ZIP エントリ作成に失敗しました: %v\n", path, err)
			}
			return nil
		}

		if !info.Mode().IsRegular() {
			fmt.Printf("INFO: 通常ファイルでない %s をスキップします\n", path)
			return nil
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			fmt.Printf("WARNING: %s の ZIP エントリ作成に失敗しました: %v\n", path, err)
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("WARNING: %s を開けませんでした: %v\n", path, err)
			return nil
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			fmt.Printf("WARNING: %s の書き込みに失敗しました: %v\n", path, err)
			return nil
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("ZIP ファイルの作成中にエラーが発生しました: %v", err)
	}

	return nil
}

// Homebrew 管轄のアプリケーション群をバックアップ
func backupHomebrew(backupDir string) error {
	cmd := exec.Command("brew", "bundle", "dump")
	cmd.Dir = backupDir
	return cmd.Run()
}

// dotfile 群をバックアップ
func backupDotfiles(backupDir string) error {
	dotfilesDir := filepath.Join(backupDir, "dotfiles")
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		return err
	}

	homeDir := os.ExpandEnv("$HOME")
	entries, err := os.ReadDir(homeDir)
	if err != nil {
		return err
	}

	excludeMap := make(map[string]bool)
	for _, file := range excludeFiles {
		excludeMap[file] = true
	}

	// デフォルトで除外するファイル
	excludeMap[".CFUserTextEncoding"] = true
	excludeMap[".Trash"] = true
	excludeMap[".DS_Store"] = true

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") {
			continue
		}
		if excludeMap[name] {
			continue
		}

		srcPath := filepath.Join(homeDir, name)
		dstPath := filepath.Join(dotfilesDir, name)

		if err := copyPath(srcPath, dstPath); err != nil {
			fmt.Printf("WARNING: %s のコピーに失敗しました: %v\n", name, err)
			continue
		}
	}

	return nil
}

// ユーザーファイル群のバックアップ
func backupUserFiles(backupDir, dirName string) error {
	userfilesDir := filepath.Join(backupDir, "userfiles", dirName)
	if err := os.MkdirAll(userfilesDir, 0755); err != nil {
		return err
	}

	srcDir := filepath.Join(os.ExpandEnv("$HOME"), dirName)
	if err := copyPath(srcDir, userfilesDir); err != nil {
		fmt.Printf("WARNING: %s ディレクトリのバックアップ中にエラーが発生しました: %v\n", dirName, err)
	}
	return nil
}

// ディレクトリやファイルを再帰的にコピー
func copyPath(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// シンボリックリンクの処理
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(linkTarget, dst)
	}

	if srcInfo.IsDir() {
		// ディレクトリの作成
		if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
			return err
		}

		// ディレクトリ内のファイルを再帰的にコピー
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			srcPath := filepath.Join(src, entry.Name())
			dstPath := filepath.Join(dst, entry.Name())

			if err := copyPath(srcPath, dstPath); err != nil {
				fmt.Printf("WARNING: %sのコピーに失敗しました: %v\n", entry.Name(), err)
				continue
			}
		}
		return nil
	}

	return copyFile(src, dst)
}

// ファイルをコピー
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("ソースファイルを開けません: %v", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("ファイル情報の取得に失敗しました: %v", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("出力ファイルの作成に失敗しました: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("ファイルのコピーに失敗しました: %v", err)
	}

	return nil
}
