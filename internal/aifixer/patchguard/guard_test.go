package patchguard

import "testing"

// ─── Round-1 happy tests (preserved verbatim) ─────────────────────────

func TestValidateValidCode(t *testing.T) {
	code := "fun main() { println(\"hello\") }"
	if err := Validate(code); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateUnbalancedBraces(t *testing.T) {
	code := "fun main() { println(\"hello\")"
	err := Validate(code)
	if err == nil {
		t.Fatal("expected error for unbalanced braces")
	}
	if err.Error() != "unbalanced braces: net count is 1" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateUnbalancedParentheses(t *testing.T) {
	code := "fun main( { println(\"hello\") }"
	err := Validate(code)
	if err == nil {
		t.Fatal("expected error for unbalanced parentheses")
	}
}

// ─── Round-2 task #4: lexer-aware edge cases ──────────────────────────

// TestValidateStringWithBraces valida que braces literales dentro de
// strings NO cuentan. Caso del round-1 caso feliz: el block `{ val s = "{"
// } fun other() { val t = "}" }` es global balanced, con braces literales
// `{` y `}` dentro de los strings.
func TestValidateStringWithBraces(t *testing.T) {
	code := `{ val s = "{" } fun other() { val t = "}" }`
	if err := Validate(code); err != nil {
		t.Fatalf("string braces should be ignored: %v", err)
	}
}

// TestValidateUnbalancedBracesInsideStringIgnored es el regression guard
// CRÍTICO del round-1 → round-2 fix. El código tiene `{` sin cerrar dentro
// de un string; el round-1 patchguard contaría ese `{` y reportaría
// "unbalanced braces". El lexer-aware ignora contenido de strings.
func TestValidateUnbalancedBracesInsideStringIgnored(t *testing.T) {
	code := `val s = "{" /* hasta aquí sin braces reales */`
	if err := Validate(code); err != nil {
		t.Fatalf("{ inside string should be ignored (lexer regression), got: %v", err)
	}
}

// TestValidateRawStringWithBraces: raw strings no procesan escapes y
// pueden contener braces desbalanceados sin contar.
func TestValidateRawStringWithBraces(t *testing.T) {
	code := `val raw = """ { not balanced in raw """ + x`
	if err := Validate(code); err != nil {
		t.Fatalf("raw string braces should be ignored: %v", err)
	}
}

// TestValidateCommentLine: line comments `//` consumen hasta newline Y
// braces/parens dentro son ignorados.
func TestValidateCommentLine(t *testing.T) {
	code := "fun foo() { // { ( ignored\n}"
	if err := Validate(code); err != nil {
		t.Fatalf("line comment braces/parens should be ignored: %v", err)
	}
}

// TestValidateCommentBlock: block comments `/* ... */` balancean contra
// `*/` incluso con braces desbalanceados dentro.
func TestValidateCommentBlock(t *testing.T) {
	code := "fun foo() { /* { ( ignored */ }"
	if err := Validate(code); err != nil {
		t.Fatalf("block comment braces/parens should be ignored: %v", err)
	}
}

// TestValidateCharLiteral: char literals `'x'` con escapes. Brace literal
// dentro de char (e.g. `'}'`) se ignora.
func TestValidateCharLiteral(t *testing.T) {
	code := `val a = '}'; val b = '{'`
	if err := Validate(code); err != nil {
		t.Fatalf("char literal braces should be ignored: %v", err)
	}
}

// TestValidateCharLiteralEscaped: char con escape sequence `'\\''`,
// `'\\n'`, etc. El lexer debe consumir el byte escapado atómicamente.
func TestValidateCharLiteralEscaped(t *testing.T) {
	code := `val x = '\''; val y = '\n'; val z = '\\'`
	if err := Validate(code); err != nil {
		t.Fatalf("escaped char literal should be parsed atomically: %v", err)
	}
}

// TestValidateTemplateSimple: template `${var}` dentro de un string. El
// template se abre/cierra correctamente y el net brace es 0.
func TestValidateTemplateSimple(t *testing.T) {
	code := `val s = "abc ${x} def"`
	if err := Validate(code); err != nil {
		t.Fatalf("simple template should balance: %v", err)
	}
}

// TestValidateTemplateWithLambda: caso crítico del round-1 — una lambda
// dentro de un template que introduce `{}` anidados. Net braces debe
// ser 0. La lambda abre y cierra su propio par de braces dentro del
// scope del template.
func TestValidateTemplateWithLambda(t *testing.T) {
	code := `val s = "${list.map { x -> x + 1 }.joinToString()}"`
	if err := Validate(code); err != nil {
		t.Fatalf("template with lambda should balance: %v", err)
	}
}

// TestValidateTemplateWithNestedString: dentro de template, un string
// literal (e.g. `"${\"inner\"}"`) — el lexer debe entrar/salir del
// string anidado correctamente.
func TestValidateTemplateWithNestedString(t *testing.T) {
	code := `val s = "${"inner"}"`
	if err := Validate(code); err != nil {
		t.Fatalf("template with nested string should balance: %v", err)
	}
}

// TestValidateStringWithEscape: backslash-brace dentro de string es
// literal — el `\}` no cuenta como cierre de string.
func TestValidateStringWithEscape(t *testing.T) {
	code := `val s = "\{"; val t = "\}"`
	if err := Validate(code); err != nil {
		t.Fatalf("escaped braces in string should be literal (not counted): %v", err)
	}
}

// TestValidateTemplateUnbalancedFails: confirma que el lexer NO es
// demasiado permisivo. Un template sin cerrar debe fallar.
func TestValidateTemplateUnbalancedFails(t *testing.T) {
	code := `val s = "abc ${oh no`
	err := Validate(code)
	if err == nil {
		t.Fatal("expected error for unclosed template")
	}
}

// TestValidateEmptyString: edge case — strings totalmente vacíos `""`.
// No deben afectar el conteo.
func TestValidateEmptyString(t *testing.T) {
	code := `val a = ""; val b = ""
	fun foo() { bar() }`
	if err := Validate(code); err != nil {
		t.Fatalf("empty strings should be no-op: %v", err)
	}
}

// TestValidateUnbalancedBracesInsideBlockComment: block comment con
// braces desbalanceados — el lexer debe consumir TODO el `/* ... */`
// incluyendo el contenido mal balanceado.
func TestValidateUnbalancedBracesInsideBlockComment(t *testing.T) {
	code := "fun foo() { /* { opens, close via EOF */ }"
	if err := Validate(code); err != nil {
		t.Fatalf("block comment consumes its content even if unbalanced: %v", err)
	}
}
