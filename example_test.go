package lexer_test

import (
	"fmt"

	"github.com/bmatsuo/go-lexer"
)

// This example shows usage of Advance, the lowest level Lexer method.  The
// functions IsEOF and IsInvalid can assist handling special return values.
func ExampleLexer_advance() {
	const itemOK lexer.ItemType = 0
	start := func(lex *lexer.Lexer) lexer.StateFn {
		c, n := lex.Advance()
		if lexer.IsEOF(c, n) {
			return lex.Errorf("unexpected end-of-input")
		}
		if lexer.IsInvalid(c, n) {
			return lex.Errorf("invalid utf-8 rune")
		}
		if c == '.' {
			lex.Emit(itemOK)
			return nil
		}
		return lex.Errorf("unexpected rune %q", c)
	}
	input := ""
	lex := lexer.New(start, input)
	item := lex.Next()
	err := item.Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ok")
	// Output: unexpected end-of-input
}
