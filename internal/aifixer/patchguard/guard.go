package patchguard

import "fmt"

// Validate checks that the provided Kotlin source has balanced braces { }
// and parentheses ( ), while ignoring content inside:
//
//   - Regular strings        ("...")  with escape sequences (\", \n, \\, …)
//   - Raw strings           ("""…""") without escapes
//   - Char literals          ('.')    with escape sequences
//   - Template expressions   ${…}    (counts their inner braces/parens but
//                                       exits back to the surrounding string
//                                       mode when the expression closes)
//   - Line comments         //…
//   - Block comments        /* … */
//
// Errors preserve the round-1 message format (consumed by tests and the
// `fix --ai` CLI that displayed them to users):
//
//   "unbalanced braces: net count is N"
//   "unbalanced parentheses: net count is N"
//
// Limitations (task #4 of round-2 polish):
//
//   - Templates nested 2+ levels are NOT supported (e.g. `"${\"${x}\"}"`).
//     Covers the common case `"${var}"` and `"${simple.expr}"` including
//     lambdas that introduce their own `{}`. If a future contributor needs
//     arbitrary nesting, swap `templateOpen` for a `[]int` snapshot stack.
//   - Raw strings whose CONTENT contains `"""` end prematurely. Rare in
//     real Kotlin code.
func Validate(code string) error {
	// Tracking modes. The current mode is always `stack[len(stack)-1]`.
	type mode int
	const (
		modeCode mode = iota
		modeLineComment
		modeBlockComment
		modeSingleString
		modeRawString
		modeCharLiteral
		modeTemplate
	)

	stack := []mode{modeCode}
	// templateOpen es el contador `braces` en el momento de entrar al
	// actual modeTemplate. -1 fuera de template. Si templates se anidan 2+
	// niveles, snapshot se sobrescribe; por ahora single-level.
	templateOpen := -1
	braces, parens := 0, 0
	i := 0

	// hasTripleQuote detecta `"""` desde la posición idx.
	hasTripleQuote := func(idx int) bool {
		return idx+2 < len(code) &&
			code[idx] == '"' && code[idx+1] == '"' && code[idx+2] == '"'
	}

	for i < len(code) {
		ch := code[i]
		switch stack[len(stack)-1] {
		case modeCode:
			switch {
			case hasTripleQuote(i):
				stack = append(stack, modeRawString)
				i += 3
			case i+1 < len(code) && ch == '/' && code[i+1] == '/':
				stack = append(stack, modeLineComment)
				i += 2
			case i+1 < len(code) && ch == '/' && code[i+1] == '*':
				stack = append(stack, modeBlockComment)
				i += 2
			case ch == '"':
				stack = append(stack, modeSingleString)
				i++
			case ch == '\'':
				stack = append(stack, modeCharLiteral)
				i++
			case ch == '{':
				braces++
				i++
			case ch == '}':
				braces--
				i++
			case ch == '(':
				parens++
				i++
			case ch == ')':
				parens--
				i++
			default:
				i++
			}

		case modeLineComment:
			if ch == '\n' {
				stack = stack[:len(stack)-1]
			}
			i++

		case modeBlockComment:
			if i+1 < len(code) && ch == '*' && code[i+1] == '/' {
				stack = stack[:len(stack)-1]
				i += 2
			} else {
				i++
			}

		case modeSingleString:
			switch {
			case ch == '\\' && i+1 < len(code):
				// Escape sequence: saltar `\` y el siguiente char (literal).
				i += 2
			case ch == '$' && i+1 < len(code) && code[i+1] == '{':
				// Template expression: contarlo como código entre `${` y `}`.
				// templateOpen snapshot antes del `{` para detectar el cierre.
				templateOpen = braces
				braces++
				stack = append(stack, modeTemplate)
				i += 2
			case ch == '"':
				stack = stack[:len(stack)-1]
				i++
			default:
				i++
			}

		case modeRawString:
			switch {
			case hasTripleQuote(i):
				stack = stack[:len(stack)-1]
				i += 3
			case ch == '$' && i+1 < len(code) && code[i+1] == '{':
				templateOpen = braces
				braces++
				stack = append(stack, modeTemplate)
				i += 2
			default:
				i++
			}

		case modeCharLiteral:
			switch {
			case ch == '\\' && i+1 < len(code):
				i += 2
			case ch == '\'':
				stack = stack[:len(stack)-1]
				i++
			default:
				i++
			}

		case modeTemplate:
			// Dentro del template, el código sigue reglas Kotlin estándar,
			// pero podemos encontrar strings, raw strings, char literals,
			// o comentarios anidados. Empujamos su modo correspondiente y
			// volvemos a modeTemplate al delimitador de cierre.
			switch {
			case hasTripleQuote(i):
				stack = append(stack, modeRawString)
				i += 3
			case ch == '"':
				stack = append(stack, modeSingleString)
				i++
			case ch == '\'':
				stack = append(stack, modeCharLiteral)
				i++
			case i+1 < len(code) && ch == '/' && code[i+1] == '/':
				stack = append(stack, modeLineComment)
				i += 2
			case i+1 < len(code) && ch == '/' && code[i+1] == '*':
				stack = append(stack, modeBlockComment)
				i += 2
			case ch == '{':
				braces++
				i++
			case ch == '}':
				braces--
				i++
				if templateOpen >= 0 && braces == templateOpen {
					// Cierre del template — volvemos al string que lo contenía.
					stack = stack[:len(stack)-1]
					templateOpen = -1
				}
			case ch == '(':
				parens++
				i++
			case ch == ')':
				parens--
				i++
			default:
				i++
			}
		}
	}

	if braces != 0 {
		return fmt.Errorf("unbalanced braces: net count is %d", braces)
	}
	if parens != 0 {
		return fmt.Errorf("unbalanced parentheses: net count is %d", parens)
	}
	return nil
}
