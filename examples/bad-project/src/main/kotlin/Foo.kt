package p

import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch

fun a() {
    println(123)                                    // 1. MagicNumber (compose-magic-number / generic)
    println("sk-1234567890abcdef")                  // 2. HardcodedPassword (sec-hardcoded-secret)
    GlobalScope.launch { println(42) }               // 3. GlobalCoroutineUsage (coroutine-global-scope)
}
