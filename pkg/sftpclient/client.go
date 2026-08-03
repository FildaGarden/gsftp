package sftpclient

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AuthType represents the SSH authentication mode
type AuthType int

const (
	AuthPassword AuthType = iota
	AuthPrivateKey
	AuthSSHAgent
)

// Config holding connection parameters
type Config struct {
	Host       string
	Port       int
	Username   string
	AuthType   AuthType
	Password   string
	KeyPath    string
	Passphrase string
	Timeout    time.Duration
}

// FileItem unifies local and remote file metadata for TUI display
type FileItem struct {
	Name      string
	Size      int64
	Mode      os.FileMode
	ModTime   time.Time
	IsDir     bool
	IsSymlink bool
}

// Client wraps SSH & SFTP connections
type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	Config     Config
}

// Connect establishes SSH and SFTP sessions
func Connect(cfg Config) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	authMethods := []ssh.AuthMethod{}

	switch cfg.AuthType {
	case AuthSSHAgent:
		agentAuth, err := getSSHAgentAuth()
		if err != nil {
			return nil, fmt.Errorf("SSH Agent Error: %w", err)
		}
		authMethods = append(authMethods, agentAuth)

	case AuthPrivateKey:
		if cfg.KeyPath == "" {
			home, _ := os.UserHomeDir()
			cfg.KeyPath = filepath.Join(home, ".ssh", "id_ed25519")
			if _, err := os.Stat(cfg.KeyPath); os.IsNotExist(err) {
				cfg.KeyPath = filepath.Join(home, ".ssh", "id_rsa")
			}
		}
		keyAuth, err := getKeyAuth(cfg.KeyPath, cfg.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("Private Key Error: %w", err)
		}
		authMethods = append(authMethods, keyAuth)

	case AuthPassword:
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Can be updated for known_hosts
		Timeout:         cfg.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial SSH (%s): %w", addr, err)
	}

	sftpConn, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("Failed to create SFTP client: %w", err)
	}

	return &Client{
		sshClient:  sshConn,
		sftpClient: sftpConn,
		Config:     cfg,
	}, nil
}

func getSSHAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to SSH agent socket: %w", err)
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), nil
}

func getKeyAuth(keyPath, passphrase string) (ssh.AuthMethod, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read private key file: %w", err)
	}

	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// Close terminates connection
func (c *Client) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// ReadDir returns list of FileItem for a remote path
func (c *Client) ReadDir(path string) ([]FileItem, error) {
	entries, err := c.sftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var items []FileItem
	for _, entry := range entries {
		mode := entry.Mode()
		items = append(items, FileItem{
			Name:      entry.Name(),
			Size:      entry.Size(),
			Mode:      mode,
			ModTime:   entry.ModTime(),
			IsDir:     entry.IsDir(),
			IsSymlink: mode&os.ModeSymlink != 0,
		})
	}
	return items, nil
}

// GetWD returns current remote working directory
func (c *Client) GetWD() (string, error) {
	return c.sftpClient.Getwd()
}

// Mkdir creates remote directory
func (c *Client) Mkdir(path string) error {
	return c.sftpClient.MkdirAll(path)
}

// Remove deletes remote file or empty directory
func (c *Client) Remove(path string) error {
	return c.sftpClient.Remove(path)
}

// RemoveAll deletes remote directory recursively
func (c *Client) RemoveAll(path string) error {
	return c.sftpClient.RemoveAll(path)
}

// Rename renames remote file/directory
func (c *Client) Rename(oldPath, newPath string) error {
	return c.sftpClient.Rename(oldPath, newPath)
}

// OpenRemoteFile opens remote file for reading
func (c *Client) OpenRemoteFile(path string) (io.ReadCloser, error) {
	return c.sftpClient.Open(path)
}

// CreateRemoteFile creates remote file for writing
func (c *Client) CreateRemoteFile(path string) (io.WriteCloser, error) {
	return c.sftpClient.Create(path)
}

// ReadLocalDir lists contents of local directory
func ReadLocalDir(path string) ([]FileItem, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var items []FileItem
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		mode := info.Mode()
		items = append(items, FileItem{
			Name:      entry.Name(),
			Size:      info.Size(),
			Mode:      mode,
			ModTime:   info.ModTime(),
			IsDir:     entry.IsDir(),
			IsSymlink: mode&os.ModeSymlink != 0,
		})
	}
	return items, nil
}

// CleanPath normalizes path
func CleanPath(p string) string {
	cleaned := filepath.Clean(p)
	if strings.HasPrefix(cleaned, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(cleaned, "~"))
	}
	return cleaned
}
