// Copyright 2012, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*  Filename:    lexer.go
 *  Author:      Bryan Matsuo <bryan.matsuo [at] gmail.com>
 *  Created:     2012-11-02 22:10:59.782356 -0700 PDT
 *  Description: Main source file in go-lexer
 */

// Package lexer provides a simple scanner and types for handrolling lexers.
// The implementation is based on Rob Pikes talk.
//	  	http://www.youtube.com/watch?v=HxaD_trXwRE
package lexer

import (
	"container/ring"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const EOF rune = 0x04

// A state function that scans runes from the lexer's input and emits items.
// The state functions are responsible for emitting ItemEOF.
type StateFn func(*Lexer) StateFn

type Lexer struct {
	input string     // string being scanned
	start int        // start position for the current lexeme
	pos   int        // current position
	width int        // length of the last rune read
	last  rune       // the last rune read
	state StateFn    // the current state
	items *ring.Ring // buffer of lexed items
}

// Create a new lexer. Use the lexer by instantiating it and repeatedly calling Next.
func New(start StateFn, input string) *Lexer {
	return &Lexer{input: input}
}

// The starting position of the current item.
func (l *Lexer) Start() int {
	return l.start
}

// The position following the last read rune.
func (l *Lexer) Pos() int {
	return l.pos
}

// The contents of the item currently being lexed.
func (l *Lexer) Current() string {
	return l.input[l.start:l.pos]
}

// The last rune read from the input stream.
func (l *Lexer) Last() (r rune, width int) {
	return l.last, l.width
}

// Add one rune of the input stream to the current lexeme.
func (l *Lexer) Advance() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return EOF
	}
	l.last, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return l.last
}

// Remove the last rune from the current lexeme and place back in the stream.
func (l *Lexer) Backup() {
	l.pos -= l.width
}

// Returns the next rune in the input stream without adding it to the current lexeme.
func (l *Lexer) Peek() (r rune, width int) {
	r := l.Advance()
	width = l.width
	l.Backup()
	return
}

// Throw away the current lexeme (do not call Emit).
func (l *Lexer) Ignore() {
	l.start = l.pos
}

// Advance the lexer only if the next rune is in the valid string.
func (l *Lexer) Accept(valid string) (ok bool) {
	if ok = strings.IndexRune(valid, l.Advance()) >= 0; !ok {
		l.Backup()
	}
	return
}

// Advance the lexer as long the next rune is in the valid string.
func (l *Lexer) AcceptRun(valid string) (n int) {
	for l.Accept(valid) {
		n++
	}
	return
}

// Like Advance but uses a range table for efficiency.
func (l *Lexer) AcceptRangeTable(rangeTab *unicode.RangeTable) (ok bool) {
	r := l.Advance()
	if ok = unicode.Is(rangeTab, r); !ok {
		l.Backup()
	}
	return
}

// Like AdvanceRun but uses a range table for efficiency.
func (l *Lexer) AcceptRangeTableRun(rangeTab *unicode.RangeTable) (n int) {
	for l.AcceptRangeTable(rangeTab) {
		n++
	}
	return
}

// Emit an error from the Lexer.
func (l *Lexer) Errorf(format string, args ...interface{}) StateFn {
	l.enqueue(&Item{
		ItemError,
		l.start,
		fmt.Sprintf(format, args...),
	})
	return nil
}

// Emit the current value as an Item with the specified type.
func (l *Lexer) Emit(t ItemType) {
	l.enqueue(&Item{
		t,
		l.start,
		l.input[l.start:l.pos],
	})
	l.start = l.pos
}

// The method by which items are extracted from the input.
// Can result in infinite loops if state functions fail to emit ItemEOF.
func (l *Lexer) Next() *Item {
	for l.items == nil {
		l.state = l.state(l)
	}
	return l.dequeue()
}

func (l *Lexer) enqueue(i *Item) {
	r := ring.New(1)
	r.Value = i
	if l.items == nil {
		l.items = r
	} else {
		l.items.Prev().Link(r)
	}
}

func (l *Lexer) dequeue() *Item {
	if l.items == nil {
		return nil
	}
	r := l.items.Unlink(1)
	if r == l.items {
		l.items = nil
	}
	return r.Value.(*Item)
}

// A type for all the types of items in the language being lexed.
type ItemType uint16

// Special item types.
const (
	ItemEOF ItemType = math.MaxUint16 - iota
	ItemError
)

// An individual scanned item (a lexeme).
type Item struct {
	Type  ItemType
	Pos   int
	Value string
}

func (i *Item) String() string {
	switch i.Type {
	case ItemError:
		return i.Value
	case ItemEOF:
		return "EOF"
	}
	if len(i.Value) > 10 {
		return fmt.Sprintf("%.10q...", i.Value)
	}
	return i.Value
}
