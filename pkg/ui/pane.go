package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gsftp/pkg/sftpclient"
	"gsftp/pkg/utils"

	"github.com/charmbracelet/lipgloss"
)

type SortCriterion int

const (
	SortByName SortCriterion = iota
	SortBySize
	SortByTime
)

// PaneModel represents one side of the dual-pane file manager
type PaneModel struct {
	Path           string
	Items          []sftpclient.FileItem
	Cursor         int
	Selected       map[string]bool
	ViewportOffset int
	Width          int
	Height         int
	SortBy         SortCriterion
	SortAsc        bool
	IsActive       bool
	IsRemote       bool
}

func NewPaneModel(isRemote bool) PaneModel {
	return PaneModel{
		Path:     "/",
		Items:    []sftpclient.FileItem{},
		Selected: make(map[string]bool),
		SortBy:   SortByName,
		SortAsc:  true,
		IsRemote: isRemote,
		Height:   20,
		Width:    40,
	}
}

func (p *PaneModel) SetItems(items []sftpclient.FileItem) {
	p.Items = nil
	p.Selected = make(map[string]bool)

	// Add parent directory entry ".." if not at root
	cleanPath := filepath.Clean(p.Path)
	if cleanPath != "/" && cleanPath != "." {
		p.Items = append(p.Items, sftpclient.FileItem{
			Name:  "..",
			IsDir: true,
		})
	}

	p.Items = append(p.Items, items...)
	p.SortItems()

	if p.Cursor >= len(p.Items) {
		p.Cursor = max(0, len(p.Items)-1)
	}
}

func (p *PaneModel) SortItems() {
	if len(p.Items) <= 1 {
		return
	}

	// Always keep ".." at the top
	hasParent := p.Items[0].Name == ".."
	sortSlice := p.Items
	if hasParent {
		sortSlice = p.Items[1:]
	}

	sort.SliceStable(sortSlice, func(i, j int) bool {
		a, b := sortSlice[i], sortSlice[j]

		// Directories always first
		if a.IsDir != b.IsDir {
			return a.IsDir
		}

		var result bool
		switch p.SortBy {
		case SortBySize:
			result = a.Size < b.Size
		case SortByTime:
			result = a.ModTime.Before(b.ModTime)
		default: // SortByName
			result = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}

		if !p.SortAsc {
			return !result
		}
		return result
	})
}

func (p *PaneModel) getVisibleRows() int {
	h := p.Height - 4
	if h < 1 {
		h = 15
	}
	return h
}

func (p *PaneModel) MoveCursorUp() {
	if p.Cursor > 0 {
		p.Cursor--
		if p.Cursor < p.ViewportOffset {
			p.ViewportOffset = p.Cursor
		}
	}
}

func (p *PaneModel) MoveCursorDown() {
	if p.Cursor < len(p.Items)-1 {
		p.Cursor++
		visibleRows := p.getVisibleRows()
		if p.Cursor >= p.ViewportOffset+visibleRows {
			p.ViewportOffset = p.Cursor - visibleRows + 1
		}
	}
}

func (p *PaneModel) Home() {
	p.Cursor = 0
	p.ViewportOffset = 0
}

func (p *PaneModel) End() {
	if len(p.Items) > 0 {
		p.Cursor = len(p.Items) - 1
		visibleRows := p.getVisibleRows()
		p.ViewportOffset = max(0, p.Cursor-visibleRows+1)
	}
}

func (p *PaneModel) PageUp() {
	visibleRows := p.getVisibleRows()
	p.Cursor = max(0, p.Cursor-visibleRows)
	p.ViewportOffset = max(0, p.ViewportOffset-visibleRows)
}

func (p *PaneModel) PageDown() {
	visibleRows := p.getVisibleRows()
	if len(p.Items) > 0 {
		p.Cursor = min(len(p.Items)-1, p.Cursor+visibleRows)
		p.ViewportOffset = min(max(0, len(p.Items)-visibleRows), p.ViewportOffset+visibleRows)
	}
}

func (p *PaneModel) ToggleSelect() {
	if len(p.Items) == 0 {
		return
	}
	item := p.Items[p.Cursor]
	if item.Name == ".." {
		return
	}

	if p.Selected[item.Name] {
		delete(p.Selected, item.Name)
	} else {
		p.Selected[item.Name] = true
	}
	p.MoveCursorDown()
}

func (p *PaneModel) SelectAll() {
	if len(p.Selected) > 0 {
		p.Selected = make(map[string]bool)
		return
	}
	for _, item := range p.Items {
		if item.Name != ".." {
			p.Selected[item.Name] = true
		}
	}
}

func (p *PaneModel) ClearSelection() {
	p.Selected = make(map[string]bool)
}

func (p *PaneModel) GetCurrentItem() *sftpclient.FileItem {
	if len(p.Items) == 0 || p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return nil
	}
	return &p.Items[p.Cursor]
}

func (p *PaneModel) GetSelectedOrCurrentItems() []sftpclient.FileItem {
	var result []sftpclient.FileItem
	for _, item := range p.Items {
		if item.Name != ".." && p.Selected[item.Name] {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		curr := p.GetCurrentItem()
		if curr != nil && curr.Name != ".." {
			result = append(result, *curr)
		}
	}
	return result
}

func (p *PaneModel) View() string {
	// Header bar
	badge := BadgeLocalStyle.Render(" LOCAL ")
	if p.IsRemote {
		badge = BadgeRemoteStyle.Render(" REMOTE ")
	}

	truncatedPath := utils.TruncateString(p.Path, max(10, p.Width-15))
	header := fmt.Sprintf("%s %s", badge, TitleStyle.Render(truncatedPath))

	// Height calculation for scrollable rows
	visibleRows := p.getVisibleRows()
	var rows []string
	endIndex := min(len(p.Items), p.ViewportOffset+visibleRows)

	nameColWidth := max(12, p.Width-28)

	for i := p.ViewportOffset; i < endIndex; i++ {
		item := p.Items[i]
		icon := utils.GetFileIcon(item.Name, item.IsDir, item.IsSymlink)

		nameStr := utils.TruncateString(item.Name, nameColWidth)
		sizeStr := ""
		if !item.IsDir {
			sizeStr = utils.FormatBytes(item.Size)
		} else if item.Name == ".." {
			sizeStr = "<UP>"
		} else {
			sizeStr = "<DIR>"
		}

		timeStr := ""
		if item.Name != ".." {
			timeStr = utils.FormatTime(item.ModTime)
		}

		// Formatting row
		rowContent := fmt.Sprintf("%s %-*s %9s %11s",
			icon,
			nameColWidth, nameStr,
			sizeStr,
			timeStr,
		)

		// Apply selection & cursor styling
		var rowStyle lipgloss.Style
		if i == p.Cursor && p.IsActive {
			rowStyle = CursorRowStyle
		} else if p.Selected[item.Name] {
			rowStyle = SelectedRowStyle
		} else if item.IsDir {
			rowStyle = DirItemStyle
		} else if item.IsSymlink {
			rowStyle = SymlinkItemStyle
		} else {
			rowStyle = lipgloss.NewStyle().Foreground(ColorText)
		}

		rows = append(rows, rowStyle.Render(rowContent))
	}

	// Pad empty space if fewer items
	for len(rows) < visibleRows {
		rows = append(rows, "")
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Footer info bar
	selectedCount := len(p.Selected)
	footerInfo := fmt.Sprintf("Items: %d | Selected: %d", len(p.Items), selectedCount)
	footer := MutedTextStyle.Render(footerInfo)

	panelContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)

	// Apply active vs inactive outer border style
	borderStyle := InactivePaneBorder
	if p.IsActive {
		borderStyle = ActivePaneBorder
	}

	return borderStyle.Width(p.Width).Height(p.Height).Render(panelContent)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
