# gsftp

A dual-pane SFTP client for the terminal, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Dual-pane view**: Side-by-side local and remote file navigation.
- **Keyboard & mouse support**: Vim-style keys (`h`/`j`/`k`/`l`), standard arrows, function keys (`F5`–`F8`), and mouse scrolling.
- **File management**: Upload, download, rename, delete (with confirmation), and create directories.
- **Transfer progress**: Real-time progress bar during uploads and downloads.
- **Sorting**: Toggle sorting by name, size, or modification time.

## Build & Run

### Prerequisites

- Go 1.22 or newer

### Build from source

```bash
git clone https://github.com/FildaGarden/gsftp.git
cd gsftp
go build -o gsftp main.go
```

### Usage

```bash
./gsftp
```

Enter your host, port, username, and password or SSH private key path in the initial prompt to connect.

## Keybindings

| Key | Action |
| --- | --- |
| `Tab` | Switch active pane (Local ↔ Remote) |
| `j` / `k` or `↓` / `↑` | Move cursor down / up |
| `h` / `l` or `←` / `→` | Parent directory / Enter directory |
| `Space` / `v` | Select / Deselect file |
| `a` / `Ctrl+A` | Select all files |
| `F5` / `u` | Upload |
| `F6` / `d` | Download |
| `F7` / `n` | Create directory |
| `F8` / `x` / `Del` | Delete file/directory |
| `r` | Rename |
| `s` | Toggle sort (Name → Size → Time) |
| `?` | Toggle help overlay |
| `q` | Quit |

## License

MIT
