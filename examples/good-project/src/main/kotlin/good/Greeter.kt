// Greeter.kt is the canonical "clean" example used in scoring-fixtures/good.json.
// kdoctor expects a Health Score 95-100 here. We deliberately keep this file
// minimal: no GlobalScope, no hardcoded passwords, no unused imports, no god
// class, no private dead code. Any findings here are bugs in either the
// rule mapping, the smoke test scoring band, or kdoctor's baseline.

package good

class Greeter(private val name: String) {
    fun greet(): String = "Hello, $name!"
}

fun main() {
    println(Greeter("World").greet())
}
