package shellquote

import (
	"fmt"
	"io"
	"sync"
	"unicode/utf8"
)

type token struct {
	toktype int
	chrs    []rune
}

const (
	state_done = 1 << iota
	state_squote
	state_dquote
	state_backslash
)

func is_delim(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func tokenize_all(in <-chan rune) ([]token, error) {
	var tok token
	var result []token
	for {
		tok = token{}
		st := tokenize_start(in, &tok)
		if st == -1 {
			return result, io.ErrUnexpectedEOF
		} else if st == 1 {
			if len(tok.chrs) > 0 || tok.toktype != 0 {
				result = append(result, tok)
			}
			return result, nil
		}
		result = append(result, tok)
	}
}

func tokenize_start(in <-chan rune, tok *token) int {
	for {
		ch := <-in
		if is_delim(ch) && (len(tok.chrs) > 0 || tok.toktype != 0) {
			return 0
		} else if ch == '\'' {
			if !tokenize_squote(in, tok) {
				return -1
			}
		} else if ch == '"' {
			if !tokenize_dquote(in, tok) {
				return -1
			}
		} else if ch == 0 {
			return 1
		} else {
			tok.chrs = append(tok.chrs, ch)
		}
	}
}

func tokenize_squote(in <-chan rune, tok *token) bool {
	tok.toktype |= state_squote
	for {
		ch := <-in
		if ch == '\'' {
			tok.toktype &^= state_squote
			return true
		} else if ch == 0 {
			return false
		}
		tok.chrs = append(tok.chrs, ch)
	}
}

func tokenize_dquote(in <-chan rune, tok *token) bool {
	tok.toktype |= state_dquote
	for {
		ch := <-in
		if ch == '\\' {
			if !tokenize_backslash(in, tok) {
				return false
			}
		} else if ch == '"' {
			tok.toktype &^= state_dquote
			return true
		} else if ch == 0 {
			return false
		} else {
			tok.chrs = append(tok.chrs, ch)
		}
	}
}

func tokenize_backslash(in <-chan rune, tok *token) bool {
	tok.toktype |= state_backslash
	ch := <-in
	if ch == 0 {
		return false
	}
	tok.chrs = append(tok.chrs, ch)
	tok.toktype &^= state_backslash
	return true
}

func tokenize_one(in []byte) token {
	return token{}
	//ch := make(chan rune)
	//var tok token
	//var wg sync.WaitGroup
	//var is_done bool
	//wg.Add(1)
	//go func() {
	//	is_done = tokenize_start(ch, &tok)
	//	wg.Done()
	//}()
	//
	//for p := 0; p < len(in); {
	//	r, runeLen := utf8.DecodeRune(in[p:])
	//	p += runeLen
	//	fmt.Println("sending", string(r))
	//	ch <- r
	//}
	//close(ch)
	//wg.Wait()
	//return tok
}

func FullTokenize(in []byte) ([]string, error) {
	ch := make(chan rune)
	var tokens []token
	var err error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		tokens, err = tokenize_all(ch)
		wg.Done()
	}()

	for p := 0; p < len(in); {
		r, runeLen := utf8.DecodeRune(in[p:])
		p += runeLen
		fmt.Println("sending", string(r))
		ch <- r
	}
	close(ch)
	wg.Wait()

	var result = make([]string, len(tokens))
	for i, v := range tokens {
		result[i] = string(v.chrs)
	}
	return result, err
}
