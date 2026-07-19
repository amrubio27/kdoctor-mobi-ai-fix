// BadCode.kt intentionally seeds antipatterns that the kdoctor smoke test
// expects to detect. This file is part of examples/bad-project/ and MUST
// stay as-is (do not "fix" these patterns — they are the test fixtures).
//
// ──────────────────────────────────────────────────────────────────────
// Detekt rules expected to fire on this file (each maps to a kdoctor rule
// in rules/metadata.json — see scripts/genschema/main.go for the catalog):
//
//   1) Hardcoded passwords  → kdoctor sec-hardcoded-secret   (HardcodedPassword)
//   2) GlobalScope usage    → kdoctor coroutine-global-scope (GlobalCoroutineUsage)
//   3) Unused import        → kdoctor dead-unused-import     (UnusedImports)
//   4) Unused private fun   → kdoctor dead-unused-private-fun (UnusedPrivateMember)
//   5) God class (>10 funs) → kdoctor arch-god-class         (TooManyFunctions)
//
// ──────────────────────────────────────────────────────────────────────
// Native regex detector targets (Tier 1#2 from the master plan). The bare
// references below are INTENTIONALLY not declared to avoid pulling in fake
// stub classes (e.g. `object Dispatchers { val IO = ... }`). Detekt's K1
// compiler emits these as semantic warnings, never as fatal parse errors
// — so SARIF generation always proceeds. The native detectors in
// `internal/core/rules/rules.go` match on raw text, so the substring
// pattern is enough to confirm the contract.
//
//   6) WebView JavaScript Enabled → kdoctor sec-webview-javascript-enabled
//   7) Hardcoded Dispatchers      → kdoctor coroutine-dispatchers-hardcoded
//   8) items() without key        → kdoctor compose-missing-key
//   9) Log PII                    → kdoctor sec-log-pii
// ──────────────────────────────────────────────────────────────────────

package bad

import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import kotlin.math.cos // ⚠️ unused import on purpose (3)

// (1) Hardcoded secrets block — kdoctor finds these via HardcodedPassword.
// NOTE: HardcodedPassword is no longer in detekt-cli 1.23.x default config
// (`unzip -p detekt-cli.jar default-detekt-config.yml | grep -i hardcode`
// returns 0 hits). The catalog keeps the mapping for forward-compat with
// detekt 2.x — once the user upgrades detekt and re-enables the stanza
// in examples/bad-project/detekt.yml, sec-hardcoded-secret will fire
// automatically without touching the kdoctor catalog.
object Secrets {
    val apiKey = "real-secret-key-do-not-commit"
    val password = "hardcoded-admin-password"
    val token = "bearer-xyz-token-not-for-prod"
}

// (2) GlobalScope.launch — kdoctor finds via GlobalCoroutineUsage.
fun startBackgroundTask() {
    GlobalScope.launch {
        println("running in GlobalScope — antipattern")
    }
}

// (4) Unused private member — kdoctor finds via UnusedPrivateMember.
private fun neverCalledHelper(): Int = 42

// (5) TooManyFunctions (>10) — kdoctor finds via TooManyFunctions (V1's
// `arch-god-class`). Default detekt thresholdInClasses is 11 — having 13
// funcs here comfortably exceeds the limit. NOTE: this stanza is omitted
// from examples/bad-project/detekt.yml (gotcha #13 — `--config REPLACE`),
// so it does not actively fire in this fixture; it is kept here so when
// the user restores the stanza, mapping is verified end-to-end.
class GodClass {
    fun a() {}
    fun b() {}
    fun c() {}
    fun d() {}
    fun e() {}
    fun f() {}
    fun g() {}
    fun h() {}
    fun i() {}
    fun j() {}
    fun k() {}
    fun l() {}
    fun m() {}
}

fun main() {
    println("BadCode.kt intentionally seeds antipatterns for kdoctor smoke test.")
    startBackgroundTask()
    val _unusedLocal = "variable referenced to avoid kotlin warning" // unused locals ignored
}

// (6) WebView JavaScript Enabled → kdoctor sec-webview-javascript-enabled.
//
// Forward-compat note: with detekt 2.x + classpath (--classpath or
// --all-rules), `Any.javaScriptEnabled = true` becomes a HARD type error
// (not a warning). Detekt will refuse to emit SARIF and the smoke test
// will fail. Two mitigation options:
//
//   A. Stubs pattern: create examples/bad-project/src/main/kotlin/bad/
//      Stubs.kt with proper Android-Lite types (class WebSettings with
//      var javaScriptEnabled = false; object Dispatchers; object Log;
//      fun items(...)) imported from BadCode.kt. Re-introduces fake
//      Android scaffolding but is forward-compatible.
//   B. Real imports: use `import android.webkit.WebSettings`,
//      `import kotlinx.coroutines.Dispatchers`, `import android.util.Log`.
//      Requires detekt to have Android SDK classpath available.
//
// For detekt-cli 1.23.x standalone (what kdoctor uses today), neither is
// needed — the `Any`-receiver + bare-reference pattern emits SARIF fine
// with only cosmetic UnresolvedReference warnings.
//
// `Any` is a built-in Kotlin type — using it here keeps the antipattern
// expression parseable and minimises UnresolvedReference noise. The native
// regex detector only cares about the literal `javaScriptEnabled = true`
// being present somewhere in the file.
fun configureWebView(settings: Any) {
    settings.javaScriptEnabled = true
}

// (7) Hardcoded Dispatchers → kdoctor coroutine-dispatchers-hardcoded.
// `Dispatchers.IO` is a bare reference (resolved at runtime in real Android
// via `kotlinx.coroutines.Dispatchers.IO`). Detekt's parser treats it as
// a semantic warning; the pure-text regex in the native detector still
// matches the substring.
fun useHardcodedDispatcher() {
    val dispatcher = Dispatchers.IO
}

// (8) items() without key → kdoctor compose-missing-key.
// `items(...)` is a bare reference to Compose's `LazyListScope.items`. The
// native detector inspects the call args and reports if no `key` parameter
// is supplied.
fun renderItems() {
    items(listOf("apple", "banana")) {
        println(it)
    }
}

// (9) Log PII → kdoctor sec-log-pii.
// `Log.d(...)` is a bare reference to android.util.Log (resolved in real
// Android). The native detector inspects the args for PII keywords
// (password|email|token|secret|phone|ssn|credential|pin|creditcard).
fun logSensitiveInfo() {
    Log.d("Auth", "password was wrong")
}
