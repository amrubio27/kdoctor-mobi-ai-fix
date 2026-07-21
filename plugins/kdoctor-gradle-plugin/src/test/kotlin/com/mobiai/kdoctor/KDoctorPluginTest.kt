package com.mobiai.kdoctor

import org.gradle.testkit.runner.GradleRunner
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.io.File

class KDoctorPluginTest {

    @TempDir
    lateinit var testProjectDir: File

    @Test
    fun `plugin registers task and runs successfully`() {
        // Setup build file
        val buildFile = File(testProjectDir, "build.gradle.kts")
        buildFile.writeText("""
            plugins {
                id("com.mobiai.kdoctor")
            }
            
            kdoctor {
                executable.set("echo") // just echo something so it succeeds on any machine
                failBelow.set(0)
            }
        """.trimIndent())

        // Run the task
        val result = GradleRunner.create()
            .withProjectDir(testProjectDir)
            .withArguments("kdoctorScan")
            .withPluginClasspath()
            .build()

        assertTrue(result.output.contains("BUILD SUCCESSFUL"))
    }
}
