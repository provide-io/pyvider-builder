package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func compileRuntime(outPath string) error {
	buildDir, err := os.MkdirTemp("", "pyvider-runtime-build-")
	if err != nil { return err }
	defer os.RemoveAll(buildDir)

	runtimeTemplateDir := "launcher"
	files, err := os.ReadDir(runtimeTemplateDir)
	if err != nil { return err }
	for _, file := range files {
		srcPath := filepath.Join(runtimeTemplateDir, file.Name())
		dstPath := filepath.Join(buildDir, file.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil { return err }
		if err := os.WriteFile(dstPath, data, 0644); err != nil { return err }
	}

	buildCmd := exec.Command("go", "build", "-o", outPath, ".")
	buildCmd.Dir = buildDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil { return err }
	log.Printf("   - ✅ Go runtime compiled successfully to: %s\n", outPath)
	return nil
}
