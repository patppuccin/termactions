package termactions

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Choice represents a single selectable item in a [Select] or [MultiSelect] prompt.
type Choice struct {
	Value string
	Label string
}

type selectionNav struct {
	cursorIdx int
	startIdx  int
	endIdx    int
	pageSize  int
}

func (n *selectionNav) up(total int) {
	if n.cursorIdx > 0 {
		n.cursorIdx--
		if n.cursorIdx < n.startIdx {
			n.startIdx = n.cursorIdx
			n.endIdx = min(n.startIdx+n.pageSize, total)
		}
	}
}

func (n *selectionNav) down(total int) {
	if n.cursorIdx < total-1 {
		n.cursorIdx++
		if n.cursorIdx >= n.endIdx {
			n.endIdx = n.cursorIdx + 1
			n.startIdx = max(0, n.endIdx-n.pageSize)
		}
	}
}

func (n *selectionNav) reset(total, pageSize int) {
	n.pageSize = pageSize
	if total == 0 {
		n.cursorIdx, n.startIdx, n.endIdx = 0, 0, 0
		return
	}
	if n.cursorIdx >= total {
		n.cursorIdx = total - 1
	}
	n.startIdx = max(0, n.cursorIdx-n.pageSize+1)
	n.endIdx = min(n.startIdx+n.pageSize, total)
}

func renderSelectionChoice(c Choice, cur, sel bool, printableWidth int, cursorIndicator, selectionMarker string, styles *StyleMap) string {
	cursorWidth := runewidth.StringWidth(cursorIndicator)
	selWidth := runewidth.StringWidth(selectionMarker)
	cursorSpacer := strings.Repeat(" ", cursorWidth)
	selSpacer := strings.Repeat(" ", selWidth)
	label := TruncToWidth(c.Label, printableWidth-(cursorWidth+selWidth+1))
	switch {
	case sel && cur:
		return safeStyle(styles.SelectionItemSelectedMarker).Sprint(cursorIndicator+selectionMarker) + " " +
			safeStyle(styles.SelectionItemSelectedLabel).Sprint(label)
	case sel:
		return cursorSpacer +
			safeStyle(styles.SelectionItemSelectedMarker).Sprint(selectionMarker) + " " +
			safeStyle(styles.SelectionItemSelectedLabel).Sprint(label)
	case cur:
		return safeStyle(styles.SelectionItemCurrentMarker).Sprint(cursorIndicator) + selSpacer + " " +
			safeStyle(styles.SelectionItemCurrentLabel).Sprint(label)
	default:
		return cursorSpacer + selSpacer + " " +
			safeStyle(styles.SelectionItemNormalLabel).Sprint(label)
	}
}

func filterSelectionChoices(choices []Choice, query string) []Choice {
	if query == "" {
		return choices
	}
	var filtered []Choice
	q := strings.ToLower(query)
	for _, c := range choices {
		if strings.Contains(strings.ToLower(c.Label), q) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
