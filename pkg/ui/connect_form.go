package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gsftp/pkg/sftpclient"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FormField int

const (
	FieldHost FormField = iota
	FieldPort
	FieldUser
	FieldAuthType
	FieldPassword
	FieldKeyPath
)

type ConnectFormModel struct {
	inputs   []textinput.Model
	focused  FormField
	authType sftpclient.AuthType
	Err      string
	Width    int
	Height   int
}

func NewConnectFormModel() ConnectFormModel {
	inputs := make([]textinput.Model, 6)

	// Host input
	inputs[FieldHost] = textinput.New()
	inputs[FieldHost].Placeholder = "localhost or example.com"
	inputs[FieldHost].Focus()
	inputs[FieldHost].CharLimit = 128
	inputs[FieldHost].Width = 32

	// Port input
	inputs[FieldPort] = textinput.New()
	inputs[FieldPort].SetValue("22")
	inputs[FieldPort].CharLimit = 5
	inputs[FieldPort].Width = 10

	// User input
	inputs[FieldUser] = textinput.New()
	inputs[FieldUser].Placeholder = "username"
	if currentUser := os.Getenv("USER"); currentUser != "" {
		inputs[FieldUser].SetValue(currentUser)
	}
	inputs[FieldUser].Width = 24

	// Auth Type dummy input (handled via Left/Right arrows)
	inputs[FieldAuthType] = textinput.New()

	// Password input
	inputs[FieldPassword] = textinput.New()
	inputs[FieldPassword].Placeholder = "Password / Key Passphrase"
	inputs[FieldPassword].EchoMode = textinput.EchoPassword
	inputs[FieldPassword].EchoCharacter = '•'
	inputs[FieldPassword].Width = 32

	// Key Path input
	inputs[FieldKeyPath] = textinput.New()
	home, _ := os.UserHomeDir()
	defaultKey := filepath.Join(home, ".ssh", "id_ed25519")
	if _, err := os.Stat(defaultKey); os.IsNotExist(err) {
		defaultKey = filepath.Join(home, ".ssh", "id_rsa")
	}
	inputs[FieldKeyPath].SetValue(defaultKey)
	inputs[FieldKeyPath].Width = 36

	return ConnectFormModel{
		inputs:   inputs,
		focused:  FieldHost,
		authType: sftpclient.AuthPrivateKey,
	}
}

func (m ConnectFormModel) GetConfig() (sftpclient.Config, error) {
	host := strings.TrimSpace(m.inputs[FieldHost].Value())
	if host == "" {
		return sftpclient.Config{}, fmt.Errorf("Host name is required")
	}

	port, err := strconv.Atoi(m.inputs[FieldPort].Value())
	if err != nil || port <= 0 || port > 65535 {
		port = 22
	}

	user := strings.TrimSpace(m.inputs[FieldUser].Value())
	if user == "" {
		return sftpclient.Config{}, fmt.Errorf("Username is required")
	}

	return sftpclient.Config{
		Host:       host,
		Port:       port,
		Username:   user,
		AuthType:   m.authType,
		Password:   m.inputs[FieldPassword].Value(),
		KeyPath:    m.inputs[FieldKeyPath].Value(),
		Passphrase: m.inputs[FieldPassword].Value(),
	}, nil
}

func (m ConnectFormModel) Update(msg tea.Msg) (ConnectFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.nextField()
			return m, nil
		case "shift+tab", "up":
			m.prevField()
			return m, nil
		case "left", "right":
			if m.focused == FieldAuthType {
				if m.authType == sftpclient.AuthSSHAgent {
					m.authType = sftpclient.AuthPrivateKey
				} else if m.authType == sftpclient.AuthPrivateKey {
					m.authType = sftpclient.AuthPassword
				} else {
					m.authType = sftpclient.AuthSSHAgent
				}
				return m, nil
			}
		}
	}

	// Update active textinput
	var cmd tea.Cmd
	if m.focused != FieldAuthType {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}
	return m, cmd
}

func (m *ConnectFormModel) nextField() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + 1) % 6
	if m.focused != FieldAuthType {
		m.inputs[m.focused].Focus()
	}
}

func (m *ConnectFormModel) prevField() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + 5) % 6
	if m.focused != FieldAuthType {
		m.inputs[m.focused].Focus()
	}
}

func (m ConnectFormModel) View() string {
	authStr := "Private Key"
	switch m.authType {
	case sftpclient.AuthSSHAgent:
		authStr = "SSH Agent"
	case sftpclient.AuthPassword:
		authStr = "Password"
	}

	var formRows []string
	formRows = append(formRows, TitleStyle.Render("GARDEN SFTP CLIENT "))
	formRows = append(formRows, "")

	labels := []string{"Host Server:", "SSH Port:", "Username:", "Auth Mode:", "Password / Key Passphrase:", "Private Key Path:"}

	for i := 0; i < 6; i++ {
		field := FormField(i)
		lblStyle := lipgloss.NewStyle().Width(26).Bold(field == m.focused).Foreground(ColorText)
		if field == m.focused {
			lblStyle = lblStyle.Foreground(ColorSecondary)
		}

		var inputView string
		if field == FieldAuthType {
			inputView = fmt.Sprintf("◀ %s ▶ (Use ← / → keys)", lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Render(authStr))
		} else {
			inputView = m.inputs[i].View()
		}

		formRows = append(formRows, fmt.Sprintf("%s %s", lblStyle.Render(labels[i]), inputView))
	}

	if m.Err != "" {
		formRows = append(formRows, "")
		formRows = append(formRows, StatusErrorStyle.Render("⚠️ Error: "+m.Err))
	}

	formRows = append(formRows, "")
	formRows = append(formRows, HelpDescStyle.Render("Press [Enter] to Connect • [Tab/Up/Down] Navigate • [Esc/Ctrl+C] Quit"))

	dialog := ModalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, formRows...))

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, dialog)
}
