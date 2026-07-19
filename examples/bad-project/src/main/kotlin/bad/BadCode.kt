// BadCode.kt intentionally seeds antipatterns that the kdoctor smoke test
// expects to detect. This file is part of examples/bad-project/ and MUST
// stay as-is (do not "fix" these patterns — they are the test fixtures).
//
// Detekt rules expected to fire on this file (each maps to a kdoctor rule
// in rules/metadata.json — see feat commit ee191e0 for the catalog):
//
//   1) Hardcoded passwords  → kdoctor sec-hardcoded-secret       (HardcodedPassword)
//   2) GlobalScope usage    → kdoctor coroutine-global-scope     (GlobalCoroutineUsage)
//   3) Unused import        → kdoctor dead-unused-import          (UnusedImport)
//   4) Unused private fun   → kdoctor dead-unused-private-fun    (UnusedPrivateMember)
//   5) God class (>10 funs) → kdoctor arch-god-class             (TooManyFunctions)
//
// Plus extras by default-detekt Phase 1.5 mappings (FunctionNaming,
// MagicNumber etc.) — these are bonuses; the smoke test only asserts
// the explicit mustIncludeFinding list in scoring-fixtures/bad.json.

package bad

import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import kotlin.math.cos // ⚠️ unused import on purpose (3)

// (1) Hardcoded secrets block — kdoctor finds these via HardcodedPassword.
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

// (5) TooManyFunctions (>10) — kdoctor finds via TooManyFunctions in V1's
// `arch-god-class`. Default detekt thresholdInClasses is 11 — having 13
// funcs here comfortably exceeds the limit.
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
