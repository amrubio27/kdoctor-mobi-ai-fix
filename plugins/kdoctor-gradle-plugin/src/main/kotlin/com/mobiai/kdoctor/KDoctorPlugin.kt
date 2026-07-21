package com.mobiai.kdoctor

import org.gradle.api.Plugin
import org.gradle.api.Project
import org.gradle.api.provider.Property
import org.gradle.api.tasks.Exec

interface KDoctorExtension {
    val executable: Property<String>
    val failBelow: Property<Int>
    val baseline: Property<String>
    val diff: Property<String>
}

class KDoctorPlugin : Plugin<Project> {
    override fun apply(project: Project) {
        val extension = project.extensions.create("kdoctor", KDoctorExtension::class.java).apply {
            executable.convention("kdoctor")
            failBelow.convention(0)
            baseline.convention("")
            diff.convention("")
        }

        project.tasks.register("kdoctorScan", KDoctorScanTask::class.java) {
            group = "verification"
            description = "Runs kdoctor scan to compute Health Score"

            val extExecutable = extension.executable.get()
            executable = extExecutable
            
            val argsList = mutableListOf("scan", "--type=android", "--project-dir", project.rootDir.absolutePath)
            
            val fb = extension.failBelow.get()
            if (fb > 0) {
                argsList.add("--fail-below=$fb")
            }
            
            val bl = extension.baseline.get()
            if (bl.isNotEmpty()) {
                argsList.add("--baseline=$bl")
            }
            
            val df = extension.diff.get()
            if (df.isNotEmpty()) {
                argsList.add("--diff=$df")
            }

            args = argsList
        }
    }
}
