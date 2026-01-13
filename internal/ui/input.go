package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func ReadKey() (string, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	b := make([]byte, 3)
	n, err := os.Stdin.Read(b)
	if err != nil {
		return "", err
	}

	if n == 3 && b[0] == 27 && b[1] == 91 {
		switch b[2] {
		case 65:
			return "UP", nil
		case 66:
			return "DOWN", nil
		case 67:
			return "RIGHT", nil
		case 68:
			return "LEFT", nil
		}
	}

	if b[0] == 3 {
		os.Exit(0)
	}

	return string(b[:n]), nil
}

func Select(title string, options []string) int {
	selectedIndex := 0

	for {
		ClearScreen()

		var content []string
		content = append(content, "Use Arrow Keys to Select, ENTER to Confirm")
		content = append(content, "")

		for i, opt := range options {
			prefix := "   "
			if i == selectedIndex {
				prefix = fmt.Sprintf("%s > %s", Yellow, Reset)
				opt = fmt.Sprintf("%s%s%s", Bold, opt, Reset)
			}
			content = append(content, fmt.Sprintf("%s%s", prefix, opt))
		}

		DrawCenteredBox(title, content)

		key, _ := ReadKey()
		switch key {
		case "UP":
			if selectedIndex > 0 {
				selectedIndex--
			}
		case "DOWN":
			if selectedIndex < len(options)-1 {
				selectedIndex++
			}
		case "\r", "\n":
			return selectedIndex
		}
	}
}

func Input(title, prompt string, isPassword bool) string {
	ClearScreen()

	width := GetTerminalWidth()
	boxWidth := 60
	if boxWidth > width {
		boxWidth = width - 4
	}
	leftPad := (width - boxWidth) / 2
	padding := strings.Repeat(" ", leftPad)

	fmt.Printf("%s%s╭─ %s %s╮%s\n", padding, Cyan, White+title+Cyan, strings.Repeat("─", boxWidth-len(title)-5), Reset)

	fmt.Printf("%s%s│ %s%s%s│%s\n", padding, Cyan, Reset+prompt, strings.Repeat(" ", boxWidth-len(prompt)-4), Cyan, Reset)

	fmt.Printf("%s%s│ %s%s│%s\n", padding, Cyan, strings.Repeat(" ", boxWidth-4), Cyan, Reset)

	inputLineViz := "> __________________________________________________"
	fmt.Printf("%s%s│ %s%s%s│%s\n", padding, Cyan, Gray+inputLineViz, strings.Repeat(" ", boxWidth-len(inputLineViz)-4), Cyan, Reset)

	fmt.Printf("%s%s│ %s%s│%s\n", padding, Cyan, strings.Repeat(" ", boxWidth-4), Cyan, Reset)

	footer := "Type and press ENTER"
	fmt.Printf("%s%s│ %s%s%s│%s\n", padding, Cyan, Gray+footer, strings.Repeat(" ", boxWidth-len(footer)-4), Cyan, Reset)

	fmt.Printf("%s%s╰%s╯%s\n", padding, Cyan, strings.Repeat("─", boxWidth-2), Reset)

	fmt.Printf("\033[4A")

	fmt.Printf("\r")
	fmt.Printf("\033[%dC", leftPad+4)

	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var input []rune

	fmt.Print("\033[?25h")
	defer fmt.Print("\033[?25h")

	for {
		b := make([]byte, 1)
		os.Stdin.Read(b)
		char := b[0]

		if char == 3 {
			term.Restore(int(os.Stdin.Fd()), oldState)
			os.Exit(0)
		}
		if char == 13 || char == 10 {
			break
		}

		if char == 127 || char == 8 {
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
			continue
		}

		if char >= 32 && char <= 126 {

			if len(input) < 50 {
				input = append(input, rune(char))
				if isPassword {
					fmt.Print("*")
				} else {
					fmt.Print(string(char))
				}
			}
		}
	}

	return string(input)
}
