// Build mínimo de examples/bad-project. Sólo lo necesario para que
// `./gradlew detekt` produzca SARIF y kdoctor pueda parsearlo.
//
// NO es un proyecto Android completo: lo abstraemos a Kotlin-only
// porque detekt no requiere Android Manifest ni SDK. Si el usuario
// quiere probar contra un proyecto Android real, basta con apuntar
// `kdoctor scan --type=kmp / --type=android` a su proyecto.

plugins {
    kotlin("jvm") version "1.9.22"
    id("io.gitlab.arturmichalczak.detekt") version "1.23.6"
}

repositories {
    mavenCentral()
}

dependencies {
    implementation("io.gitlab.arturmichalczak.detekt:detekt-formatting:1.23.6")
    implementation("com.facebook.kotlin.fresco:detekt-rules-compose:0.5.0")
    detektPlugins("io.gitlab.arturmichalczak.detekt:detekt-formatting:1.23.6")
    detektPlugins("com.facebook.kotlin.fresco:detekt-rules-compose:0.5.0")
}

detekt {
    toolVersion = "1.23.6"
    source.setFrom("src/main/kotlin")
    config.from("detekt.yml")
}
