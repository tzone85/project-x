package dashboard

import (
	"fmt"
	"strings"
)

// ScrollableViewport is a simple scrollable content viewer.
// It stores content as a string, splits it by lines, and displays
// a visible slice based on the current offset and height.
type ScrollableViewport struct {
	content    string
	lines      []string
	offset     int
	height     int
	totalLines int
}

// NewScrollableViewport creates a viewport with sensible defaults.
func NewScrollableViewport() *ScrollableViewport {
	return &ScrollableViewport{
		height: 10,
	}
}

// SetContent replaces the viewport content and recomputes line count.
// The offset is clamped to remain valid for the new content.
func (v *ScrollableViewport) SetContent(content string) {
	v.content = content
	if content == "" {
		v.lines = nil
		v.totalLines = 0
	} else {
		v.lines = strings.Split(content, "\n")
		v.totalLines = len(v.lines)
	}
	v.clampOffset()
}

// SetHeight sets the visible height in lines.
func (v *ScrollableViewport) SetHeight(h int) {
	if h < 1 {
		h = 1
	}
	v.height = h
	v.clampOffset()
}

// ScrollDown moves the viewport down by one line.
func (v *ScrollableViewport) ScrollDown() {
	v.offset++
	v.clampOffset()
}

// ScrollUp moves the viewport up by one line.
func (v *ScrollableViewport) ScrollUp() {
	v.offset--
	v.clampOffset()
}

// GotoTop scrolls to the top of the content.
func (v *ScrollableViewport) GotoTop() {
	v.offset = 0
}

// GotoBottom scrolls to the bottom of the content.
func (v *ScrollableViewport) GotoBottom() {
	v.offset = v.maxOffset()
}

// PageUp moves the viewport up by one page.
func (v *ScrollableViewport) PageUp() {
	v.offset -= v.height
	v.clampOffset()
}

// PageDown moves the viewport down by one page.
func (v *ScrollableViewport) PageDown() {
	v.offset += v.height
	v.clampOffset()
}

// View returns the visible slice of content as a string.
func (v *ScrollableViewport) View() string {
	if v.totalLines == 0 {
		return ""
	}

	end := v.offset + v.height
	if end > v.totalLines {
		end = v.totalLines
	}

	visible := v.lines[v.offset:end]
	return strings.Join(visible, "\n")
}

// ScrollIndicator returns a string like "[3/15]" showing the current
// scroll position relative to total lines.
func (v *ScrollableViewport) ScrollIndicator() string {
	if v.totalLines == 0 {
		return "[0/0]"
	}

	currentLine := v.offset + 1
	return fmt.Sprintf("[%d/%d]", currentLine, v.totalLines)
}

// AtTop returns true if the viewport is scrolled to the top.
func (v *ScrollableViewport) AtTop() bool {
	return v.offset == 0
}

// AtBottom returns true if the viewport is scrolled to the bottom.
func (v *ScrollableViewport) AtBottom() bool {
	return v.offset >= v.maxOffset()
}

// maxOffset returns the maximum valid offset.
func (v *ScrollableViewport) maxOffset() int {
	max := v.totalLines - v.height
	if max < 0 {
		return 0
	}
	return max
}

// clampOffset ensures the offset is within valid bounds.
func (v *ScrollableViewport) clampOffset() {
	if v.offset < 0 {
		v.offset = 0
	}
	max := v.maxOffset()
	if v.offset > max {
		v.offset = max
	}
}
