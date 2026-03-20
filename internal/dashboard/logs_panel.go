package dashboard

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const maxLogLines = 100

// renderLogs reads the tail of a log file and displays recent lines.
// If the log file does not exist or cannot be read, a placeholder is shown.
func renderLogs(logPath string) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Logs"))
	b.WriteString("\n\n")

	if logPath == "" {
		b.WriteString(dimStyle.Render("  No log file configured"))
		b.WriteString("\n")
		return b.String()
	}

	lines, err := tailFile(logPath, maxLogLines)
	if err != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Cannot read log file: %s", err)))
		b.WriteString("\n")
		return b.String()
	}

	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("  Log file is empty"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(dimStyle.Render(fmt.Sprintf("  (showing last %d lines from %s)", len(lines), logPath)))
	b.WriteString("\n\n")

	for _, line := range lines {
		b.WriteString("  " + colorizeLogLine(line))
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// tailFile reads the last N lines from a file.
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var allLines []string
	scanner := bufio.NewScanner(f)

	// Use a reasonable buffer size for log lines.
	const maxLineSize = 64 * 1024
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)

	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(allLines) <= n {
		return allLines, nil
	}
	return allLines[len(allLines)-n:], nil
}

// colorizeLogLine adds basic color to log lines based on level keywords.
func colorizeLogLine(line string) string {
	upper := strings.ToUpper(line)

	switch {
	case strings.Contains(upper, "ERROR"):
		return errorStyle.Render(line)
	case strings.Contains(upper, "WARN"):
		return warnStyle.Render(line)
	case strings.Contains(upper, "DEBUG"):
		return dimStyle.Render(line)
	default:
		return line
	}
}
