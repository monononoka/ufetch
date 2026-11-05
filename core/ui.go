package core

import (
	"fmt"
	"strings"
)

const progressBarWidth = 24

// FormatProgress renders an ASCII progress line. A negative expected size means
// the server did not report the total length.
func FormatProgress(downloaded, expected int64) string {
	if downloaded < 0 {
		downloaded = 0
	}
	if expected <= 0 {
		return fmt.Sprintf("[size unknown] %s downloaded", FormatBytes(downloaded))
	}

	fraction := float64(downloaded) / float64(expected)
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction * progressBarWidth)
	if downloaded >= expected {
		filled = progressBarWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", progressBarWidth-filled)
	return fmt.Sprintf("[%s] %5.1f%%  %s / %s", bar, fraction*100, FormatBytes(downloaded), FormatBytes(expected))
}

// FormatBytes formats a byte count for compact terminal output.
func FormatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	for _, unit := range []string{"KB", "MB", "GB", "TB"} {
		value /= 1024
		if value < 1024 || unit == "TB" {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", n)
}
