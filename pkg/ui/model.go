package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"gsftp/pkg/sftpclient"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AppState int

const (
	StateConnectPrompt AppState = iota
	StateMainView
	StateTransferModal
	StateConfirmModal
	StateInputModal
	StateHelpModal
)

type ModalType int

const (
	ModalMkdir ModalType = iota
	ModalRename
	ModalDeleteConfirm
)

type Model struct {
	State          AppState
	connectForm    ConnectFormModel
	localPane      PaneModel
	remotePane     PaneModel
	activeIsRemote bool
	sftpClient     *sftpclient.Client
	transfer       TransferModel
	statusMsg      string
	isStatusErr    bool
	programRef     *tea.Program

	// Prompt Modal
	modalType   ModalType
	modalInput  textinput.Model
	modalPrompt string
	modalTarget sftpclient.FileItem

	// Terminal Dimensions
	width  int
	height int
}

func NewModel() Model {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}

	local := NewPaneModel(false)
	local.Path = home
	local.IsActive = true

	remote := NewPaneModel(true)
	remote.Path = "/"
	remote.IsActive = false

	input := textinput.New()
	input.Width = 30

	m := Model{
		State:          StateConnectPrompt,
		connectForm:    NewConnectFormModel(),
		localPane:      local,
		remotePane:     remote,
		activeIsRemote: false,
		transfer:       NewTransferModel(),
		modalInput:     input,
	}
	return m
}

func (m *Model) SetProgramRef(p *tea.Program) {
	m.programRef = p
}

func (m *Model) GetActivePane() *PaneModel {
	if m.activeIsRemote {
		return &m.remotePane
	}
	return &m.localPane
}

func (m *Model) switchActivePane() {
	m.activeIsRemote = !m.activeIsRemote
	m.localPane.IsActive = !m.activeIsRemote
	m.remotePane.IsActive = m.activeIsRemote
}

func (m Model) Init() tea.Cmd {
	// Initialize local directory listing
	return tea.Batch(
		textinput.Blink,
		ReadDirCmd(nil, m.localPane.Path, false),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.connectForm.Width = msg.Width
		m.connectForm.Height = msg.Height
		m.transfer.Width = msg.Width
		m.transfer.Height = msg.Height

		// Update pane dimensions (50% split - margins)
		paneWidth := max(20, (msg.Width-4)/2)
		paneHeight := max(10, msg.Height-4)
		m.localPane.Width = paneWidth
		m.localPane.Height = paneHeight
		m.remotePane.Width = paneWidth
		m.remotePane.Height = paneHeight
		return m, nil

	case ReadDirSuccessMsg:
		if msg.IsRemote {
			m.remotePane.Path = msg.Path
			m.remotePane.SetItems(msg.Items)
		} else {
			m.localPane.Path = msg.Path
			m.localPane.SetItems(msg.Items)
		}
		return m, nil

	case ReadDirErrMsg:
		m.setStatus(fmt.Sprintf("Failed to load directory (%s): %v", msg.Path, msg.Err), true)
		return m, nil

	case TransferProgressMsg:
		m.State = StateTransferModal
		cmd := m.transfer.UpdateProgress(msg.Transferred)
		return m, cmd

	case progress.FrameMsg:
		newProgressModel, cmd := m.transfer.ProgressBar.Update(msg)
		if pm, ok := newProgressModel.(progress.Model); ok {
			m.transfer.ProgressBar = pm
		}
		return m, cmd

	case TransferCompleteMsg:
		m.transfer.Active = false
		m.State = StateMainView
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Transfer failed for %s: %v", msg.Filename, msg.Err), true)
		} else {
			m.setStatus(fmt.Sprintf("Successfully transferred %s!", msg.Filename), false)
		}
		// Refresh both panes
		return m, tea.Batch(
			ReadDirCmd(m.sftpClient, m.localPane.Path, false),
			ReadDirCmd(m.sftpClient, m.remotePane.Path, true),
		)

	case StatusMsg:
		m.setStatus(msg.Message, msg.IsError)
		return m, nil
	}

	// Route updates depending on active AppState
	switch m.State {

	case StateConnectPrompt:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				cfg, err := m.connectForm.GetConfig()
				if err != nil {
					m.connectForm.Err = err.Error()
					return m, nil
				}

				m.setStatus("Connecting to SFTP server...", false)
				client, err := sftpclient.Connect(cfg)
				if err != nil {
					m.connectForm.Err = err.Error()
					m.setStatus("Connection failed", true)
					return m, nil
				}

				m.sftpClient = client
				m.State = StateMainView
				m.setStatus("Connected successfully!", false)

				// Load initial remote directory
				wd, err := client.GetWD()
				if err != nil || wd == "" {
					wd = "/"
				}
				return m, ReadDirCmd(m.sftpClient, wd, true)

			case "esc", "ctrl+c":
				return m, tea.Quit
			}
		}

		var cmd tea.Cmd
		m.connectForm, cmd = m.connectForm.Update(msg)
		return m, cmd

	case StateMainView:
		activePane := m.GetActivePane()

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				if m.sftpClient != nil {
					m.sftpClient.Close()
				}
				return m, tea.Quit

			case "tab":
				m.switchActivePane()
				return m, nil

			case "up", "k":
				activePane.MoveCursorUp()
				return m, nil

			case "down", "j":
				activePane.MoveCursorDown()
				return m, nil

			case "home", "g":
				activePane.Home()
				return m, nil

			case "end", "G":
				activePane.End()
				return m, nil

			case "pgup", "ctrl+u":
				activePane.PageUp()
				return m, nil

			case "pgdown", "ctrl+d":
				activePane.PageDown()
				return m, nil

			case "space", "v":
				activePane.ToggleSelect()
				return m, nil

			case "a", "ctrl+a":
				activePane.SelectAll()
				return m, nil

			case "enter", "right", "l":
				item := activePane.GetCurrentItem()
				if item != nil && item.IsDir {
					var nextPath string
					if item.Name == ".." {
						nextPath = filepath.Dir(activePane.Path)
					} else {
						nextPath = filepath.Join(activePane.Path, item.Name)
					}
					return m, ReadDirCmd(m.sftpClient, nextPath, activePane.IsRemote)
				}

			case "backspace", "left", "h":
				parentPath := filepath.Dir(activePane.Path)
				if parentPath != activePane.Path {
					return m, ReadDirCmd(m.sftpClient, parentPath, activePane.IsRemote)
				}

			case "f5", "u":
				// Upload (Local -> Remote)
				if m.sftpClient == nil {
					m.setStatus("Not connected to SFTP", true)
					return m, nil
				}
				items := m.localPane.GetSelectedOrCurrentItems()
				if len(items) == 0 {
					return m, nil
				}

				item := items[0]
				localPath := filepath.Join(m.localPane.Path, item.Name)
				remotePath := filepath.Join(m.remotePane.Path, item.Name)

				m.transfer.Start("job1", item.Name, item.Size, true)
				m.State = StateTransferModal
				return m, TransferFileCmd(m.sftpClient, localPath, remotePath, true, m.programRef)

			case "f6", "d":
				// Download (Remote -> Local)
				if m.sftpClient == nil {
					m.setStatus("Not connected to SFTP", true)
					return m, nil
				}
				items := m.remotePane.GetSelectedOrCurrentItems()
				if len(items) == 0 {
					return m, nil
				}

				item := items[0]
				remotePath := filepath.Join(m.remotePane.Path, item.Name)
				localPath := filepath.Join(m.localPane.Path, item.Name)

				m.transfer.Start("job1", item.Name, item.Size, false)
				m.State = StateTransferModal
				return m, TransferFileCmd(m.sftpClient, localPath, remotePath, false, m.programRef)

			case "f7", "n":
				// New Directory
				m.modalType = ModalMkdir
				m.modalPrompt = "Enter New Directory Name:"
				m.modalInput.Reset()
				m.modalInput.Focus()
				m.State = StateInputModal
				return m, textinput.Blink

			case "r":
				// Rename
				item := activePane.GetCurrentItem()
				if item == nil || item.Name == ".." {
					return m, nil
				}
				m.modalTarget = *item
				m.modalType = ModalRename
				m.modalPrompt = fmt.Sprintf("Rename '%s' to:", item.Name)
				m.modalInput.SetValue(item.Name)
				m.modalInput.Focus()
				m.State = StateInputModal
				return m, textinput.Blink

			case "f8", "x", "delete":
				// Delete confirmation
				item := activePane.GetCurrentItem()
				if item == nil || item.Name == ".." {
					return m, nil
				}
				m.modalTarget = *item
				m.modalType = ModalDeleteConfirm
				m.modalPrompt = fmt.Sprintf("Are you sure you want to delete '%s'?", item.Name)
				m.State = StateConfirmModal
				return m, nil

			case "s":
				// Cycle Sort
				activePane.SortBy = (activePane.SortBy + 1) % 3
				activePane.SortItems()
				return m, nil

			case "?":
				m.State = StateHelpModal
				return m, nil
			}
		}

	case StateInputModal:
		activePane := m.GetActivePane()

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				val := m.modalInput.Value()
				m.State = StateMainView
				if val == "" {
					return m, nil
				}

				targetPath := filepath.Join(activePane.Path, val)
				if m.modalType == ModalMkdir {
					if activePane.IsRemote {
						err := m.sftpClient.Mkdir(targetPath)
						if err != nil {
							m.setStatus("Mkdir failed: "+err.Error(), true)
						}
					} else {
						err := os.MkdirAll(targetPath, 0755)
						if err != nil {
							m.setStatus("Mkdir failed: "+err.Error(), true)
						}
					}
				} else if m.modalType == ModalRename {
					oldPath := filepath.Join(activePane.Path, m.modalTarget.Name)
					if activePane.IsRemote {
						err := m.sftpClient.Rename(oldPath, targetPath)
						if err != nil {
							m.setStatus("Rename failed: "+err.Error(), true)
						}
					} else {
						err := os.Rename(oldPath, targetPath)
						if err != nil {
							m.setStatus("Rename failed: "+err.Error(), true)
						}
					}
				}

				return m, ReadDirCmd(m.sftpClient, activePane.Path, activePane.IsRemote)

			case "esc":
				m.State = StateMainView
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.modalInput, cmd = m.modalInput.Update(msg)
		return m, cmd

	case StateConfirmModal:
		activePane := m.GetActivePane()

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y", "enter":
				m.State = StateMainView
				targetPath := filepath.Join(activePane.Path, m.modalTarget.Name)
				if activePane.IsRemote {
					var err error
					if m.modalTarget.IsDir {
						err = m.sftpClient.RemoveAll(targetPath)
					} else {
						err = m.sftpClient.Remove(targetPath)
					}
					if err != nil {
						m.setStatus("Delete failed: "+err.Error(), true)
					}
				} else {
					err := os.RemoveAll(targetPath)
					if err != nil {
						m.setStatus("Delete failed: "+err.Error(), true)
					}
				}
				return m, ReadDirCmd(m.sftpClient, activePane.Path, activePane.IsRemote)

			case "n", "N", "esc":
				m.State = StateMainView
				return m, nil
			}
		}

	case StateHelpModal:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "?" || msg.String() == "q" {
				m.State = StateMainView
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) setStatus(msg string, isErr bool) {
	m.statusMsg = msg
	m.isStatusErr = isErr
}

func (m Model) View() string {
	if m.State == StateConnectPrompt {
		return m.connectForm.View()
	}

	// Main Dual-Pane View
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.localPane.View(),
		m.remotePane.View(),
	)

	// Status Bar
	statusStyle := StatusSuccessStyle
	if m.isStatusErr {
		statusStyle = StatusErrorStyle
	}
	statusView := statusStyle.Render(m.statusMsg)

	// Help Bar Footer
	helpItems := []string{
		HelpKeyStyle.Render("[Tab]") + HelpDescStyle.Render(" Switch Pane"),
		HelpKeyStyle.Render("[j/k/↑/↓]") + HelpDescStyle.Render(" Move"),
		HelpKeyStyle.Render("[Space/v]") + HelpDescStyle.Render(" Select"),
		HelpKeyStyle.Render("[F5/u]") + HelpDescStyle.Render(" Upload"),
		HelpKeyStyle.Render("[F6/d]") + HelpDescStyle.Render(" Download"),
		HelpKeyStyle.Render("[F7/n]") + HelpDescStyle.Render(" Mkdir"),
		HelpKeyStyle.Render("[F8/x]") + HelpDescStyle.Render(" Delete"),
		HelpKeyStyle.Render("[r]") + HelpDescStyle.Render(" Rename"),
		HelpKeyStyle.Render("[s]") + HelpDescStyle.Render(" Sort"),
		HelpKeyStyle.Render("[?]") + HelpDescStyle.Render(" Help"),
		HelpKeyStyle.Render("[q]") + HelpDescStyle.Render(" Quit"),
	}

	footerHelp := lipgloss.JoinHorizontal(lipgloss.Top, intersperse(helpItems, " • ")...)
	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		panes,
		statusView,
		footerHelp,
	)

	// Overlay Modals if active
	if m.State == StateTransferModal {
		return m.transfer.View()
	} else if m.State == StateInputModal {
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			TitleStyle.Render(" "+m.modalPrompt+" "),
			"",
			m.modalInput.View(),
			"",
			HelpDescStyle.Render("Press [Enter] Confirm • [Esc] Cancel"),
		)
		dialog := ModalStyle.Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	} else if m.State == StateConfirmModal {
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			TitleStyle.Render(" ⚠️ CONFIRM ACTION "),
			"",
			StatusErrorStyle.Render(m.modalPrompt),
			"",
			HelpDescStyle.Render("Press [Y] / [Enter] to Confirm • [N] / [Esc] to Cancel"),
		)
		dialog := ModalStyle.Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	} else if m.State == StateHelpModal {
		helpText := TitleStyle.Render(" 📖 SFTP TUI CLIENT SHORTCUTS ") + "\n\n" +
			"Tab               : Switch active focus (Local <-> Remote)\n" +
			"Up / Down / k / j : Navigate list up / down\n" +
			"Home / End / g / G: Jump to top / bottom of list\n" +
			"PgUp / PgDown     : Page up / down\n" +
			"Enter / Right / l : Open directory\n" +
			"Backspace / Left / h: Go to parent directory\n" +
			"Space / v         : Select / Deselect file\n" +
			"a / Ctrl+A        : Select / Deselect all\n" +
			"F5 / u            : Upload selected file(s) to remote pane\n" +
			"F6 / d            : Download selected file(s) to local pane\n" +
			"F7 / n            : Create new directory (mkdir)\n" +
			"F8 / x / Del      : Delete file or directory\n" +
			"r                 : Rename file or directory\n" +
			"s                 : Toggle sort mode (Name -> Size -> Time)\n" +
			"?                 : Toggle this help window\n" +
			"q / Ctrl+C        : Exit application\n\n" +
			HelpDescStyle.Render("Press [Esc] or [?] to close help")

		dialog := ModalStyle.Render(helpText)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return mainContent
}

func intersperse(slice []string, sep string) []string {
	if len(slice) == 0 {
		return nil
	}
	res := make([]string, 0, len(slice)*2-1)
	for i, s := range slice {
		if i > 0 {
			res = append(res, sep)
		}
		res = append(res, s)
	}
	return res
}
