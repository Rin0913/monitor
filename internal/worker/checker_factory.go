package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Rin0913/monitor/internal/scheduler"
	"go.yaml.in/yaml/v4"
)

type CheckerConfig struct {
	Checkers map[string]CheckerEntry `yaml:"checkers"`
}

type CheckerEntry struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command"`
	Method     string `yaml:"method"`
	Path       string `yaml:"path"`
	TimeoutSec int    `yaml:"timeout_sec"`
}

func (e *Engine) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg CheckerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	for name, entry := range cfg.Checkers {
		switch entry.Type {
		case "command":
			e.MakeCommandChecker(name, entry.Command)
			log.Printf("Load command `%s`: %s\n", name, entry.Command)
		}
	}

	return nil
}

func (e *Engine) MakeCommandChecker(name string, command string) {
	fn := func(ctx context.Context, job *scheduler.CheckJob) (string, int, map[string]interface{}, error) {
		start := time.Now()

		cmdStr := fmt.Sprintf("%s %s", command, job.Address)
		cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		latency := int(time.Since(start) / time.Millisecond)

		data := map[string]interface{}{
			"command": cmdStr,
			"stdout":  stdout.String(),
			"stderr":  stderr.String(),
		}

		if err != nil {
			data["error"] = err.Error()
			return "DOWN", latency, data, err
		}

		return "UP", latency, data, nil
	}

	e.RegisterChecker(name, fn)
}
