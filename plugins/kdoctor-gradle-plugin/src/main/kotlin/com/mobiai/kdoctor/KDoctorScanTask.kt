package com.mobiai.kdoctor

import org.gradle.api.tasks.Exec
import org.gradle.process.ExecResult

abstract class KDoctorScanTask : Exec() {
    init {
        // By default, do not throw on non-zero exit code so we can handle it or let the build fail
        isIgnoreExitValue = false
    }
}
