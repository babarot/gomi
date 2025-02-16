package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

var (
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	alterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0F080")) // yellow

	containerStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#CCCCCC"))
)

func printWarningDeprecated(fieldName string, info *Deprecation) {
	// header
	warningMessage := warningStyle.Render("Warning: ") +
		fmt.Sprintf("Field '%s' is deprecated", fieldName)

	var messages []string

	if info != nil {
		if info.Alternative != "" {
			messages = append(
				messages,
				fmt.Sprintf("Please use '%s' instead", alterStyle.Render(info.Alternative)),
			)
		}
		if !info.DeprecatedAt.IsZero() || !info.RemovalDate.IsZero() {
			messages = append(messages, ".")
		}
		if !info.DeprecatedAt.IsZero() {
			deprecatedMsg := infoStyle.Render(
				fmt.Sprintf("Deprecated since: %s", info.DeprecatedAt.Format("2006-01-02")),
			)
			messages = append(messages, deprecatedMsg)
		}
		if !info.RemovalDate.IsZero() {
			removalMsg := infoStyle.Render(
				lo.Ternary(
					time.Now().After(info.RemovalDate),
					fmt.Sprintf("Removed at: %s", info.RemovalDate.Format("2006-01-02")),
					fmt.Sprintf("Planned removal date: %s", info.RemovalDate.Format("2006-01-02")),
				),
			)
			messages = append(messages, removalMsg)
		}
	}

	var fullMessage string
	if len(messages) > 0 {
		fullMessage = fmt.Sprintf("%s\n%s",
			warningMessage,
			lipgloss.JoinVertical(lipgloss.Left, messages...),
		)
	} else {
		fullMessage = warningMessage
	}

	fmt.Println(containerStyle.Render(fullMessage))
}

func printErrorDeprecated(fieldName string, info Deprecation) {
	// header
	errorMessage := errorStyle.Render("Error: ") +
		fmt.Sprintf("Field '%s' is already retired", fieldName)

	messages := []string{
		fmt.Sprintf("Please use '%s' instead", alterStyle.Render(info.Alternative)),
		".",
		infoStyle.Render(
			lo.Ternary(
				time.Now().After(info.RemovalDate),
				"",
				fmt.Sprintf("Deprecated on: %s", info.DeprecatedAt.Format("2006-01-02")),
			),
		),
		infoStyle.Render(
			lo.Ternary(
				time.Now().After(info.RemovalDate),
				fmt.Sprintf("Removed at: %s", info.RemovalDate.Format("2006-01-02")),
				fmt.Sprintf("Will be removed on: %s", info.RemovalDate.Format("2006-01-02")),
			),
		),
	}

	messages = filterEmptyStyledStrings(messages)

	fullMessage := fmt.Sprintf("%s\n%s",
		errorMessage,
		lipgloss.JoinVertical(lipgloss.Left, messages...),
	)

	fmt.Println(containerStyle.Render(fullMessage))
}

func isStyleRenderEffectivelyEmpty(styledStr string) bool {
	ansiEscapeRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	cleanStr := ansiEscapeRegex.ReplaceAllString(styledStr, "")

	cleanStr = strings.TrimFunc(cleanStr, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})

	return cleanStr == ""
}

func filterEmptyStyledStrings(styledStrings []string) []string {
	return lo.Filter(styledStrings, func(str string, _ int) bool {
		return !isStyleRenderEffectivelyEmpty(str)
	})
}
