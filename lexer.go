// Copyright 2012, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*  Filename:    lexer.go
 *  Author:      Bryan Matsuo <bryan.matsuo [at] gmail.com>
 *  Created:     2012-11-02 22:10:59.782356 -0700 PDT
 *  Description: Main source file in go-lexer
 */

// Package lexer provides a simple scanner and types for handrolling lexers.
// The implementation is based on Rob Pike's talk.
//	  	http://www.youtube.com/watch?v=HxaD_trXwRE
package lexer

import (
	"container/list"
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

// A type for building lexers.
type Lexer struct {
	input string     // string being scanned
	start int        // start position for the current lexeme
	pos   int        // current position
	width int        // length of the last rune read
	last  rune       // the last rune read
	state StateFn    // the current state
	items *list.List // Buffer of lexed items
}

// Create a new lexer. Must be given a non-nil state.
func New(start StateFn, input string) *Lexer {
	if start == nil {
		panic("nil start state")
	}
	return &Lexer{
		state: start,
		input: input,
		items: list.New(),
	}
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

// Add one rune of the input stream to the current lexeme. Invalid UTF-8
// codepoints cause the current call and all subsequent calls to return
// (utf8.RuneError, 1).
func (l *Lexer) Advance() (rune, int) {
	if l.pos >= len(l.input) {
		l.width = 0
		return EOF, l.width
	}
	l.last, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	if l.last == utf8.RuneError && l.width == 1 {
		return l.last, l.width
	}
	l.pos += l.width
	return l.last, l.width
}

// Remove the last rune from the current lexeme and place back in the stream.
func (l *Lexer) Backup() {
	l.pos -= l.width
}

// Returns the next rune in the input stream without adding it to the current lexeme.
func (l *Lexer) Peek() (c rune, width int) {
	defer func() { l.Backup() }()
	return l.Advance()
}

// Throw away the current lexeme (do not call Emit).
func (l *Lexer) Ignore() {
	l.start = l.pos
}

// The Accept[Run][Range] family of methods take a set and advance the lexer
// if incoming runes are in the set. The AcceptRun subfamily advance the lexer
// as far as possible.
func (l *Lexer) Accept(valid string) (ok bool) {
	r, _ := l.Advance()
	ok = strings.IndexRune(valid, r) >= 0
	if !ok {
		l.Backup()
	}
	return
}
func (l *Lexer) AcceptRange(rangeTab *unicode.RangeTable) (ok bool) {
	r, _ := l.Advance()
	ok = unicode.Is(rangeTab, r)
	if !ok {
		l.Backup()
	}
	return
}

// AcceptString advances the lexer len(s) bytes if the next len(s) bytes equal
// s. AcceptString returns true if l advanced.
func (l *Lexer) AcceptString(s string) (ok bool) {
	if strings.HasPrefix(l.input[l.pos:], s) {
		l.pos += len(s)
		return true
	}
	return false
}

func (l *Lexer) AcceptRun(valid string) (n int) {
	for l.Accept(valid) {
		n++
	}
	return
}
func (l *Lexer) AcceptRunRange(rangeTab *unicode.RangeTable) (n int) {
	for l.AcceptRange(rangeTab) {
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
// Returns nil if the lexer has entered a nil state.
func (l *Lexer) Next() (i *Item) {
	for {
		if head := l.dequeue(); head != nil {
			return head
		}
		if l.state == nil {
			return &Item{ItemEOF, l.start, ""}
		}
		l.state = l.state(l)
	}
	panic("unreachable")
}

func (l *Lexer) enqueue(i *Item) {
	l.items.PushBack(i)
}

func (l *Lexer) dequeue() *Item {
	head := l.items.Front()
	if head == nil {
		return nil
	}
	return l.items.Remove(head).(*Item)
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
