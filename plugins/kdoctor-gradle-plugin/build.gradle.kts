plugins {
    `kotlin-dsl`
    `java-gradle-plugin`
}

group = "com.mobiai.kdoctor"
version = "1.0.0-SNAPSHOT"

repositories {
    mavenCentral()
}

dependencies {
    testImplementation(kotlin("test"))
    testImplementation(gradleTestKit())
}

gradlePlugin {
    plugins {
        create("kdoctor") {
            id = "com.mobiai.kdoctor"
            implementationClass = "com.mobiai.kdoctor.KDoctorPlugin"
        }
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}
