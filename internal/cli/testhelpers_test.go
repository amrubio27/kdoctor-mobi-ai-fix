package cli

import "os"

func saveWD() (string, error) { return os.Getwd() }

func restoreWD(original string) {
	_ = os.Chdir(original)
}

func chdir(dir string) error { return os.Chdir(dir) }
