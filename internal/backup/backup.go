package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateBackup archives the data directory (excluding backups folder) into a gzipped tarball.
func CreateBackup(dataDir string) (string, error) {
	if dataDir == "" {
		dataDir = "data"
	}

	backupsDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backups directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	backupFilename := fmt.Sprintf("backup-%s.tar.gz", timestamp)
	backupPath := filepath.Join(backupsDir, backupFilename)

	outFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}

		// Skip backup directory itself to prevent recursion
		if relPath == "backups" || strings.HasPrefix(relPath, "backups"+string(filepath.Separator)) {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(relPath)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed during archive creation: %w", err)
	}

	return backupPath, nil
}
