package m

import (
	"fmt"
	"regexp"

	"github.com/walles/moar/twin"
)

type ScreenLines struct {
	inputLines             *InputLines
	firstInputLineOneBased int
	leftColumnZeroBased    int

	width  int // Display width
	height int // Display height

	searchPattern *regexp.Regexp

	showLineNumbers bool
	wrapLongLines   bool
}

// Render screen lines into an array of lines consisting of Cells.
//
// The second return value is the same as firstInputLineOneBased, but decreased
// if needed so that the end of the input is visible.
func (sl *ScreenLines) renderScreenLines() ([][]twin.Cell, int) {
	if sl.inputLines.lines == nil {
		return [][]twin.Cell{}, 0
	}

	for firstInputLineOneBased := sl.firstInputLineOneBased; firstInputLineOneBased >= sl.inputLines.firstLineOneBased; firstInputLineOneBased-- {
		rendered := sl.tryRenderScreenLines(firstInputLineOneBased)
		if len(rendered) == sl.height {
			// We managed to fill the whole screen
			return rendered, firstInputLineOneBased
		}
	}

	if sl.inputLines.firstLineOneBased == 1 {
		// We're at the top of the input document, can't go up any more, this is fine
		return sl.tryRenderScreenLines(1), 1
	}

	panic(fmt.Errorf("screen lines rendering failed, first 1-based input line available was %d", sl.inputLines.firstLineOneBased))
}

// Render screen lines into an array of lines consisting of Cells.
func (sl *ScreenLines) tryRenderScreenLines(firstInputLineOneBased int) [][]twin.Cell {
	// Count the length of the last line number
	//
	// Offsets figured out through trial-and-error...
	lastLineOneBased := sl.inputLines.firstLineOneBased + len(sl.inputLines.lines) - 1
	numberPrefixLength := len(formatNumber(uint(lastLineOneBased))) + 1
	if numberPrefixLength < 4 {
		// 4 = space for 3 digits followed by one whitespace
		//
		// https://github.com/walles/moar/issues/38
		numberPrefixLength = 4
	}

	if !sl.showLineNumbers {
		numberPrefixLength = 0
	}

	returnLines := make([][]twin.Cell, 0, sl.height)
	screenFull := false

	for lineIndex, line := range sl.inputLines.lines {
		lineNumber := sl.inputLines.firstLineOneBased + lineIndex
		if lineNumber < firstInputLineOneBased {
			// Skip this one, too early
			continue
		}

		highlighted := line.HighlightedTokens(sl.searchPattern)
		var wrapped [][]twin.Cell
		if sl.wrapLongLines {
			wrapped = wrapLine(sl.width-numberPrefixLength, highlighted)
		} else {
			// All on one line
			wrapped = [][]twin.Cell{highlighted}
		}

		for wrapIndex, inputLinePart := range wrapped {
			visibleLineNumber := &lineNumber
			if wrapIndex > 0 {
				visibleLineNumber = nil
			}

			returnLines = append(returnLines,
				sl.createScreenLine(visibleLineNumber, numberPrefixLength, inputLinePart))

			if len(returnLines) >= sl.height {
				// We have shown all the lines that can fit on the screen
				screenFull = true
				break
			}
		}

		if screenFull {
			break
		}
	}

	return returnLines
}

func (sl *ScreenLines) createScreenLine(lineNumberToShow *int, numberPrefixLength int, contents []twin.Cell) []twin.Cell {
	newLine := make([]twin.Cell, 0, sl.width)
	newLine = append(newLine, createLineNumberPrefix(lineNumberToShow, numberPrefixLength)...)

	startColumn := sl.leftColumnZeroBased
	if startColumn < len(contents) {
		endColumn := sl.leftColumnZeroBased + (sl.width - numberPrefixLength)
		if endColumn > len(contents) {
			endColumn = len(contents)
		}
		newLine = append(newLine, contents[startColumn:endColumn]...)
	}

	// Add scroll left indicator
	if sl.leftColumnZeroBased > 0 && len(contents) > 0 {
		if len(newLine) == 0 {
			// Don't panic on short lines, this new Cell will be
			// overwritten with '<' right after this if statement
			newLine = append(newLine, twin.Cell{})
		}

		// Add can-scroll-left marker
		newLine[0] = twin.Cell{
			Rune:  '<',
			Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
		}
	}

	// Add scroll right indicator
	if len(contents)+numberPrefixLength-sl.leftColumnZeroBased > sl.width {
		newLine[sl.width-1] = twin.Cell{
			Rune:  '>',
			Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
		}
	}

	return newLine
}

// Generate a line number prefix. Can be empty or all-whitespace depending on parameters.
func createLineNumberPrefix(fileLineNumber *int, numberPrefixLength int) []twin.Cell {
	if numberPrefixLength == 0 {
		return []twin.Cell{}
	}

	lineNumberPrefix := make([]twin.Cell, 0, numberPrefixLength)
	if fileLineNumber == nil {
		for len(lineNumberPrefix) < numberPrefixLength {
			lineNumberPrefix = append(lineNumberPrefix, twin.Cell{Rune: ' '})
		}
		return lineNumberPrefix
	}

	lineNumberString := formatNumber(uint(*fileLineNumber))
	lineNumberString = fmt.Sprintf("%*s ", numberPrefixLength-1, lineNumberString)
	if len(lineNumberString) > numberPrefixLength {
		panic(fmt.Errorf(
			"lineNumberString <%s> longer than numberPrefixLength %d",
			lineNumberString, numberPrefixLength))
	}

	for column, digit := range lineNumberString {
		if column >= numberPrefixLength {
			break
		}

		lineNumberPrefix = append(lineNumberPrefix, twin.NewCell(digit, _numberStyle))
	}

	return lineNumberPrefix
}
