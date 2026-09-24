package nmap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner executes nmap and returns normalized XML results.
type Runner struct {
	// Binary is the nmap executable name or path. An empty value uses nmap.
	Binary string
}

// NewRunner creates a runner using the nmap executable from PATH.
func NewRunner() Runner {
	return Runner{Binary: "nmap"}
}

// Run executes one scan. The XML output is written to a private temporary
// file so large scans do not have to be held in memory twice.
func (r Runner) Run(ctx context.Context, request Request) (Result, error) {
	started := time.Now()
	result := Result{StartedAt: started}

	binary := strings.TrimSpace(r.Binary)
	if binary == "" {
		binary = "nmap"
	}
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		return result, fmt.Errorf("find nmap executable: %w", err)
	}

	output, err := os.CreateTemp("", "lazynmap-*.xml")
	if err != nil {
		return result, fmt.Errorf("create nmap output file: %w", err)
	}
	outputPath := output.Name()
	if err := output.Close(); err != nil {
		_ = os.Remove(outputPath)
		return result, fmt.Errorf("close nmap output file: %w", err)
	}
	defer os.Remove(outputPath)

	args, err := BuildArgs(request, outputPath)
	if err != nil {
		return result, err
	}
	result.Command = append([]string{binaryPath}, args...)

	command := exec.CommandContext(ctx, binaryPath, args...)
	var stderr limitedBuffer
	stderr.limit = 1 << 20
	// Nmap writes the useful result to the XML file. Discard its normal
	// progress output so a full-port scan cannot consume unbounded memory.
	command.Stdout = io.Discard
	command.Stderr = &stderr

	runErr := command.Run()
	finished := time.Now()
	result.FinishedAt = finished
	result.Duration = finished.Sub(started)
	result.Stderr = strings.TrimSpace(stderr.String())

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if runErr != nil {
		message := strings.TrimSpace(result.Stderr)
		if message != "" {
			return result, fmt.Errorf("nmap failed: %w: %s", runErr, message)
		}
		return result, fmt.Errorf("nmap failed: %w", runErr)
	}

	scan, err := ParseXMLFile(outputPath)
	if err != nil {
		if result.Stderr != "" {
			return result, fmt.Errorf("%w; nmap stderr: %s", err, result.Stderr)
		}
		return result, err
	}
	result.Scan = scan
	if scan.Duration > 0 {
		result.Duration = scan.Duration
	}
	return result, nil
}

type limitedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = b.buffer.Write(data)
	}
	return originalLength, nil
}
