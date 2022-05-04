// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gmi

import (
	"fmt"
	"strings"
	//"unicode"
	"unicode/utf8"
)

// item represents a token or text string returned from the scanner.
type item struct {
	typ  itemType // The type of this item.
	pos  Pos      // The starting position, in bytes, of this item in the input string.
	val  string   // input which is good for this item type
	line int      // The line number at the start of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case i.typ > itemKeyword:
		return fmt.Sprintf("<%s>", i.val)
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred; value is text of error

	itemChar         // printable ASCII character; grab bag for comma etc.
	itemCharConstant // character constant

	itemEOF

	itemSpace // run of spaces separating arguments

	itemText // plain text
	itemLinkURL
	itemLinkDesc

	// Keywords appear after all the rest.
	itemKeyword // used only to delimit the keywords

	itemLink    // link prefix (=>)
	itemHeading // heading prefix (#)
	itemList    // list prefix (*)
	itemBlock   // blockquote prefix (>)
	itemPrefmt  // preformat prefix (```)
	itemNil     // the untyped nil constant, easiest to treat as a keyword

)

var key = map[string]itemType{
	"=>":  itemLink,
	"#":   itemHeading,
	"*":   itemList,
	">":   itemBlock,
	"```": itemPrefmt,
	"nil": itemNil,
}

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name  string // the name of the input; used only for error reports
	input string // the string being scanned

	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	items      chan item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
	line       int       // 1+number of newlines seen
	startLine  int       // start line of this item
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	if r == '\n' {
		l.line++
	}
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	// Correct newline count.
	if l.width == 1 && l.input[l.pos] == '\n' {
		l.line--
	}
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos], l.startLine}
	l.start = l.pos
	l.startLine = l.line
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
	l.startLine = l.line
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...), l.startLine}
	return nil
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	return <-l.items
}

// drain drains the output so the lexing goroutine will exit.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) drain() {
	for range l.items {
	}
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:      name,
		input:     input,
		items:     make(chan item),
		line:      1,
		startLine: 1,
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for state := lexPlain; state != nil; {
		state = state(l)
	}
	close(l.items)
}

// state functions
//

// scan until a line type prefix (link|prefmt), otherwise plain text
func lexPlain(l *lexer) stateFn {
	var (
		lf     int
		offset Pos
		row    string
	)
	l.width = 0
	lf = strings.Index(l.input[l.pos:], "\n")
	if lf < 0 {
		// newline is nonexistent
		l.pos = Pos(len(l.input))
		// Correctly reached EOF
		if l.pos > l.start {
			l.line += strings.Count(l.input[l.start:l.pos], "\n")
			l.emit(itemText)
		}
		l.emit(itemEOF)
		return nil
	}
	offset = Pos(lf)
	// line type is derived from first-three chars
	row = l.input[l.pos : l.pos+offset]
	if strings.HasPrefix(row, GmLink) {
		return lexLeftLink
	}
	// plain text line
	l.pos += offset
	l.emit(itemText)
	l.accept("\n")
	l.ignore()
	return lexPlain
}

//=>[<whitespace>]<URL>[<whitespace><USER-FRIENDLY LINK NAME>]
func lexLeftLink(l *lexer) stateFn {
	l.pos += Pos(len(GmLink))
	l.emit(itemLink)
	return lexLinkURL
}
func lexLinkURL(l *lexer) stateFn {
	// skip spaces for now
	l.acceptRun(" \t")
	l.ignore()
	lf := strings.Index(l.input[l.pos:], "\n")
	if lf < 0 {
		// possibly eof is the line terminator (improper syntax?)
		// fail now, ? can be more forgiving in future
		return l.errorf("Line %d does not end in newline.", l.line)
	}
	offset := Pos(lf)
	// inspect the row to right of the prefix
	remain := l.input[l.pos : l.pos+offset]
	// spaces separate the URL and friendly name
	spc := strings.IndexAny(remain, " \t")
	if spc < 0 {
		// zero spaces right of url
		l.pos += offset
		l.emit(itemLinkURL)
		l.accept("\n")
		l.ignore()
		return lexPlain
	}

	l.pos += Pos(spc)
	l.emit(itemLinkURL)

	l.acceptRun(" \t")
	l.ignore()
	// recalc row end since we advanced cursor position
	lf = strings.Index(l.input[l.pos:], "\n")
	// friendly name may be empty since it's optional
	if lf > 0 {
		l.pos += Pos(lf)
		l.emit(itemLinkDesc)
	}
	l.accept("\n")
	l.ignore()

	return lexPlain
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
