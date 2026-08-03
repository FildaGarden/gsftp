package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gsftp/pkg/sftpclient"
	"gsftp/pkg/utils"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type TransferProgressMsg struct {
	JobID       string
	Transferred int64
	Total       int64
	Filename    string
	IsUpload    bool
}

type TransferCompleteMsg struct {
	JobID    string
	Filename string
	Err      error
}

type ReadDirSuccessMsg struct {
	IsRemote bool
	Path     string
	Items    []sftpclient.FileItem
}

type ReadDirErrMsg struct {
	IsRemote bool
	Path     string
	Err      error
}

type StatusMsg struct {
	Message string
	IsError bool
}

// ProgressReader wraps io.Reader to report bytes read
type ProgressReader struct {
	reader     io.Reader
	total      int64
	copied     int64
	onProgress func(copied, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.copied += int64(n)
	if pr.onProgress != nil && n > 0 {
		pr.onProgress(pr.copied, pr.total)
	}
	return n, err
}

// TransferModel manages active transfer modal UI
type TransferModel struct {
	Active      bool
	JobID       string
	Filename    string
	IsUpload    bool
	Transferred int64
	Total       int64
	ProgressBar progress.Model
	Width       int
	Height      int
}

func NewTransferModel() TransferModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	return TransferModel{
		ProgressBar: p,
	}
}

func (t *TransferModel) Start(jobID, filename string, total int64, isUpload bool) {
	t.Active = true
	t.JobID = jobID
	t.Filename = filename
	t.Total = total
	t.Transferred = 0
	t.IsUpload = isUpload
}

func (t *TransferModel) UpdateProgress(transferred int64) tea.Cmd {
	t.Transferred = transferred
	if t.Total > 0 {
		pct := float64(transferred) / float64(t.Total)
		return t.ProgressBar.SetPercent(pct)
	}
	return nil
}

func (t TransferModel) View() string {
	if !t.Active {
		return ""
	}

	action := "⬇️ Downloading"
	if t.IsUpload {
		action = "⬆️ Uploading"
	}

	pct := float64(0)
	if t.Total > 0 {
		pct = (float64(t.Transferred) / float64(t.Total)) * 100
	}

	title := TitleStyle.Render(fmt.Sprintf(" %s: %s ", action, t.Filename))
	stats := fmt.Sprintf("%s / %s (%.1f%%)",
		utils.FormatBytes(t.Transferred),
		utils.FormatBytes(t.Total),
		pct,
	)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		t.ProgressBar.View(),
		"",
		MutedTextStyle.Render(stats),
	)

	dialog := ModalStyle.Render(content)
	return lipgloss.Place(t.Width, t.Height, lipgloss.Center, lipgloss.Center, dialog)
}

// Async Commands for File Operations & Directory Listing

func ReadDirCmd(client *sftpclient.Client, path string, isRemote bool) tea.Cmd {
	return func() tea.Msg {
		if isRemote {
			if client == nil {
				return ReadDirErrMsg{IsRemote: true, Path: path, Err: fmt.Errorf("SFTP client not connected")}
			}
			items, err := client.ReadDir(path)
			if err != nil {
				return ReadDirErrMsg{IsRemote: true, Path: path, Err: err}
			}
			return ReadDirSuccessMsg{IsRemote: true, Path: path, Items: items}
		}

		// Local read
		items, err := sftpclient.ReadLocalDir(path)
		if err != nil {
			return ReadDirErrMsg{IsRemote: false, Path: path, Err: err}
		}
		return ReadDirSuccessMsg{IsRemote: false, Path: path, Items: items}
	}
}

func TransferFileCmd(client *sftpclient.Client, localPath, remotePath string, isUpload bool, p *tea.Program) tea.Cmd {
	return func() tea.Msg {
		jobID := fmt.Sprintf("%s -> %s", localPath, remotePath)
		filename := filepath.Base(localPath)
		if !isUpload {
			filename = filepath.Base(remotePath)
		}

		if isUpload {
			// Upload local -> remote
			srcFile, err := os.Open(localPath)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			defer srcFile.Close()

			info, err := srcFile.Stat()
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			totalSize := info.Size()

			dstFile, err := client.CreateRemoteFile(remotePath)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			defer dstFile.Close()

			pr := &ProgressReader{
				reader: srcFile,
				total:  totalSize,
				onProgress: func(copied, total int64) {
					if p != nil {
						p.Send(TransferProgressMsg{
							JobID:       jobID,
							Transferred: copied,
							Total:       total,
							Filename:    filename,
							IsUpload:    true,
						})
					}
				},
			}

			_, err = io.Copy(dstFile, pr)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: nil}

		} else {
			// Download remote -> local
			srcFile, err := client.OpenRemoteFile(remotePath)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			defer srcFile.Close()

			dstFile, err := os.Create(localPath)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			defer dstFile.Close()

			// Get size for progress
			var totalSize int64 = 0
			if items, err := client.ReadDir(filepath.Dir(remotePath)); err == nil {
				for _, item := range items {
					if item.Name == filename {
						totalSize = item.Size
						break
					}
				}
			}

			pr := &ProgressReader{
				reader: srcFile,
				total:  totalSize,
				onProgress: func(copied, total int64) {
					if p != nil {
						p.Send(TransferProgressMsg{
							JobID:       jobID,
							Transferred: copied,
							Total:       total,
							Filename:    filename,
							IsUpload:    false,
						})
					}
				},
			}

			_, err = io.Copy(dstFile, pr)
			if err != nil {
				return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: err}
			}
			return TransferCompleteMsg{JobID: jobID, Filename: filename, Err: nil}
		}
	}
}
