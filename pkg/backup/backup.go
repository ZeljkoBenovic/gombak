package backup

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZeljkoBenovic/gombak/pkg/logger"
	sshclient "github.com/ZeljkoBenovic/gombak/pkg/ssh"
)

type Backup struct {
	backupDir string
	cl        *sshclient.SSH
	log       *logger.Logger

	host string
}

func New(host, port, user, pass string, log *logger.Logger) (*Backup, error) {
	cl, err := sshclient.NewSSH(
		user,
		host,
		port,
		sshclient.WithPassword(pass),
		sshclient.WithIgnoreHostKey(),
		sshclient.WithInsecureKeyExchange(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create ssh client: %w", err)
	}

	return &Backup{
		cl:  cl,
		log: log,
	}, nil
}

func (b *Backup) Close() error {
	return b.cl.Close()
}

func (b *Backup) GetRouterIdentity() (string, error) {
	b.log.Debug("Fetching system identity")

	ident, err := b.cl.Run("/system identity print")
	if err != nil {
		return "", fmt.Errorf("could not get system identity: %w", err)
	}

	var host string

	if len(ident) > 9 {
		host = strings.TrimSpace(strings.ReplaceAll(ident[8:], " ", "-"))
	} else {
		host = strings.TrimSpace(strings.ReplaceAll(ident, " ", "-"))
	}

	host = strings.ReplaceAll(host, ":", "")

	b.host = host

	return host, nil
}

func (b *Backup) RunBackup(bckDir string) error {
	b.backupDir = bckDir

	b.log.Info("Running backup", "host", b.host)

	b.log.Debug("Exporting file on the router", "cmd", "/export file=ssh-backup")

	_, err := b.cl.Run(fmt.Sprintf("/export file=ssh-backup"))
	if err != nil {
		return fmt.Errorf("could not run export: %w", err)
	}

	b.log.Debug("Creating system backup on the router", "cmd", "/system backup save name=ssh-backup")
	_, err = b.cl.Run("/system backup save name=ssh-backup")
	if err != nil {
		return fmt.Errorf("could not run system backup: %w", err)
	}

	b.log.Info("Downloading backup files", "host", b.host)

	timeNow := time.Now().Format(time.DateOnly)

	b.log.Debug("Downloading file", "name", "/ssh-backup.rsc", "host", b.host)

	if err = b.cl.Download(
		"/ssh-backup.rsc",
		path.Join(bckDir, fmt.Sprintf("%s-%s.rsc", b.host, timeNow)),
	); err != nil {
		return fmt.Errorf("could not download ssh-bakup.rsc: %w", err)
	}

	b.log.Debug("Downloading file", "name", "/ssh-backup.backup", "host", b.host)

	if err = b.cl.Download(
		"/ssh-backup.backup",
		path.Join(bckDir, fmt.Sprintf("%s-%s.backup", b.host, timeNow)),
	); err != nil {
		return fmt.Errorf("could not download ssh-backup.backup: %w", err)
	}

	b.log.Info("Backup files downloaded", "host", b.host)

	b.log.Info("Backup complete", "host", b.host)

	return nil
}

func (b *Backup) DeleteTempFiles() error {
	b.log.Info("Deleting temp backup files", "host", b.host)

	if err := b.cl.Delete("/ssh-backup.rsc"); err != nil {
		b.log.Error(
			"Backup file on the router could not be deleted",
			"err", err.Error(),
			"file_name", "/ssh-backup.rsc",
			"host", b.host,
		)
	}

	if err := b.cl.Delete("/ssh-backup.backup"); err != nil {
		b.log.Error(
			"Backup file on the router could not be deleted",
			"err", err.Error(),
			"file_name", "/ssh-backup.backup",
			"host", b.host,
		)
	}

	b.log.Info("Temp backup files deleted", "host", b.host)

	return nil
}

func RunFileCleanup(backupDir string, retentionDays int, log *logger.Logger) error {
	log.Info("Removing old backup files...")

	if err := filepath.WalkDir(backupDir, func(path string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() {
			return nil
		}

		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		now := time.Now()
		if now.Sub(fi.ModTime()).Hours() > float64(retentionDays*24) {
			log.Info("Deleting old backup file", "name", path)
			return os.Remove(path)
		}

		return nil
	}); err != nil {
		return err
	}

	log.Info("Backup files cleanup complete")

	return nil
}
