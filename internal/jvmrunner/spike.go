package jvmrunner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// RunSpike invoca un proceso JVM falso/dummy para medir el coste base
// de lanzar un proceso Java desde Go. Este es un experimento R&D (Fase 5).
func RunSpike(ctx context.Context, jarPath string) (time.Duration, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "java", "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	elapsed := time.Since(start)

	if err != nil {
		return elapsed, fmt.Errorf("JVM spike falló: %w\nSalida: %s", err, out.String())
	}

	return elapsed, nil
}
