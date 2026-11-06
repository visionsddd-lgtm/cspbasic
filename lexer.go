package main

import (
	"strconv"
	"strings"
	"unicode"
)

// --- Lexer (Supports Escaped Quotes \" and "", @/$/% Sigils, and . Dot Operator) ---

// TokenType represents the type of a lexical token.
type TokenType int
// Token types for the lexer.
const (
	TokEOF TokenType = iota
	TokNumber
	TokString
	TokIdent
	TokSemi
	TokComma
	TokColon  // :
	TokDot    // .
	TokHash   // #
	TokAssign // =
	TokLT     // <
	TokGT     // >
	TokLE     // <=
	TokGE     // >=
	TokNE     // <>
	TokPlus   // +
	TokMinus  // -
	TokStar   // *
	TokSlash  // /
	TokIDiv   // \
	TokCaret  // ^
	TokMod    // % or MOD
	TokLParen // (
	TokRParen // )
)
// Token represents a lexical token with its type and text.
type Token struct {
	Type TokenType
	Text string
}
// isTypeSigil reports whether a character is a type sigil ($, %, @, !, #).
func isTypeSigil(ch byte) bool {
	return ch == '$' || ch == '%' || ch == '@' || ch == '!' || ch == '#'
}
// LexLine lexes a single line of BASIC code into tokens. It handles numbers, strings, identifiers, operators, and comments.
func LexLine(src string) []Token {
	var tokens []Token
	i, n := 0, len(src)
	for i < n {
		ch := rune(src[i])
		if unicode.IsSpace(ch) {
			i++
			continue
		}
		if ch == '\'' {
			break
		}
		switch ch {
		case ':':
			tokens = append(tokens, Token{TokColon, ":"})
			i++
		case ';':
			tokens = append(tokens, Token{TokSemi, ";"})
			i++
		case ',':
			tokens = append(tokens, Token{TokComma, ","})
			i++
		case '.':
			if i+1 < n && unicode.IsDigit(rune(src[i+1])) {
				start := i
				i++
				for i < n && unicode.IsDigit(rune(src[i])) {
					i++
				}
				tokens = append(tokens, Token{TokNumber, src[start:i]})
			} else {
				tokens = append(tokens, Token{TokDot, "."})
				i++
			}
		case '#':
			tokens = append(tokens, Token{TokHash, "#"})
			i++
		case '=':
			tokens = append(tokens, Token{TokAssign, "="})
			i++
		case '+':
			tokens = append(tokens, Token{TokPlus, "+"})
			i++
		case '-':
			tokens = append(tokens, Token{TokMinus, "-"})
			i++
		case '*':
			tokens = append(tokens, Token{TokStar, "*"})
			i++
		case '/':
			tokens = append(tokens, Token{TokSlash, "/"})
			i++
		case '\\':
			tokens = append(tokens, Token{TokIDiv, "\\"})
			i++
		case '^':
			tokens = append(tokens, Token{TokCaret, "^"})
			i++
		case '%':
			tokens = append(tokens, Token{TokMod, "%"})
			i++
		case '(':
			tokens = append(tokens, Token{TokLParen, "("})
			i++
		case ')':
			tokens = append(tokens, Token{TokRParen, ")"})
			i++
		case '&':
			if i+2 <= n && (src[i+1] == 'H' || src[i+1] == 'h' || src[i+1] == 'O' || src[i+1] == 'o') {
				base := 16
				if src[i+1] == 'O' || src[i+1] == 'o' {
					base = 8
				}
				i += 2
				start := i
				for i < n && (unicode.IsDigit(rune(src[i])) || (base == 16 && ((src[i] >= 'a' && src[i] <= 'f') || (src[i] >= 'A' && src[i] <= 'F')))) {
					i++
				}
				val, _ := strconv.ParseInt(src[start:i], base, 64)
				tokens = append(tokens, Token{TokNumber, strconv.FormatInt(val, 10)})
			} else {
				i++
			}
		case '<':
			if i+1 < n && src[i+1] == '>' {
				tokens = append(tokens, Token{TokNE, "<>"})
				i += 2
			} else if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, Token{TokLE, "<="})
				i += 2
			} else {
				tokens = append(tokens, Token{TokLT, "<"})
				i++
			}
		case '>':
			if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, Token{TokGE, ">="})
				i += 2
			} else {
				tokens = append(tokens, Token{TokGT, ">"})
				i++
			}
		case '"':
			i++
			var sb strings.Builder
			for i < n {
				if src[i] == '\\' && i+1 < n {
					switch src[i+1] {
					case '"', '\\':
						sb.WriteByte(src[i+1])
						i += 2
						continue
					case 'n':
						sb.WriteByte('\n')
						i += 2
						continue
					case 't':
						sb.WriteByte('\t')
						i += 2
						continue
					}
				}
				if src[i] == '"' {
					if i+1 < n && src[i+1] == '"' {
						sb.WriteByte('"')
						i += 2
						continue
					}
					i++
					break
				}
				sb.WriteByte(src[i])
				i++
			}
			tokens = append(tokens, Token{TokString, sb.String()})
		default:
			if unicode.IsDigit(ch) {
				start := i
				for i < n && (unicode.IsDigit(rune(src[i])) || src[i] == '.') {
					i++
				}
				tokens = append(tokens, Token{TokNumber, src[start:i]})
			} else if unicode.IsLetter(ch) {
				start := i
				for i < n && (unicode.IsLetter(rune(src[i])) || unicode.IsDigit(rune(src[i])) || src[i] == '_') {
					i++
				}
				if i < n && isTypeSigil(src[i]) {
					i++
				}
				word := src[start:i]
				if strings.EqualFold(word, "MOD") {
					tokens = append(tokens, Token{TokMod, "MOD"})
				} else {
					tokens = append(tokens, Token{TokIdent, word})
				}
			} else {
				i++
			}
		}
	}
	return tokens
}
