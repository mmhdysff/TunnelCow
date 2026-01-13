package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

const (
	Reset  = "\033[0m"
	Cyan   = "\033[36m"
	White  = "\033[97m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Red    = "\033[31m"
	Gray   = "\033[90m"
	Bold   = "\033[1m"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

func DrawBox(title string, content []string, width int) {
	DrawBoxWithColor(title, content, width, Cyan)
}

func DrawBoxWithColor(title string, content []string, width int, color string) {

	titleLen := len(title)
	if titleLen > width-4 {
		title = title[:width-4]
		titleLen = len(title)
	}

	fmt.Printf("%s╭─ %s %s╮%s\r\n", color, White+title+color, strings.Repeat("─", width-titleLen-5), Reset)

	for _, line := range content {

		lineLen := len(stripAnsi(line))
		padding := width - lineLen - 4
		if padding < 0 {
			line = line[:width-4] + "..."
			padding = 0
		}
		fmt.Printf("%s│ %s%s %s│%s\r\n", color, Reset+line, strings.Repeat(" ", padding), color, Reset)
	}

	fmt.Printf("%s╰%s╯%s\r\n", color, strings.Repeat("─", width-2), Reset)
}

func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

func DrawCenteredBox(title string, content []string) {
	width := GetTerminalWidth()
	boxWidth := 60
	if boxWidth > width {
		boxWidth = width - 4
	}

	leftPad := (width - boxWidth) / 2
	padding := strings.Repeat(" ", leftPad)

	titleLen := len(title)
	if titleLen > boxWidth-4 {
		title = title[:boxWidth-4]
		titleLen = len(title)
	}
	fmt.Printf("%s%s╭─ %s %s╮%s\r\n", padding, Cyan, White+title+Cyan, strings.Repeat("─", boxWidth-titleLen-5), Reset)

	for _, line := range content {
		lineLen := len(stripAnsi(line))
		space := boxWidth - lineLen - 4
		if space < 0 {
			line = line[:boxWidth-4] + "..."
			space = 0
		}
		fmt.Printf("%s%s│ %s%s %s│%s\r\n", padding, Cyan, Reset+line, strings.Repeat(" ", space), Cyan, Reset)
	}

	fmt.Printf("%s%s╰%s╯%s\r\n", padding, Cyan, strings.Repeat("─", boxWidth-2), Reset)
}

func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}
