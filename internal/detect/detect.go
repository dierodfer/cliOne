// Package detect runs a tool's version command and extracts the installed
// version with the catalog regex.
package detect

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

// Timeout bounds each detect command so a hung tool never blocks the TUI.
const Timeout = 5 * time.Second

// RunDetect executes the tool's detect command and extracts the version using
// the tool's regex.
//
// stdout and stderr are matched separately, stdout first: many CLIs (npm,
// AI-assistant wrappers, etc.) print an unrelated "update available" banner
// to stderr ahead of the real version, and an unanchored regex run against
// combined output would happily latch onto the banner's version number
// instead of the tool's actual installed version. Falling back to stderr
// only when stdout has no match still supports tools like `java -version`
// that report their version on stderr alone.
//
// A missing binary yields Installed=false with no error: that is a normal
// state, not a failure. Other execution problems (timeout, non-regex-matching
// output) are reported via the result fields.
func RunDetect(ctx context.Context, def model.ToolDef) (model.DetectResult, error) {
	re, err := regexp.Compile(def.Detect.Regex)
	if err != nil {
		return model.DetectResult{Err: err}, err
	}

	fields := strings.Fields(def.Detect.Cmd)
	if len(fields) == 0 {
		err := errors.New("detect: empty command")
		return model.DetectResult{Err: err}, err
	}

	if _, lookErr := exec.LookPath(fields[0]); lookErr != nil {
		return model.DetectResult{Installed: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	res := ExtractVersion(re, stdout.String())
	if !res.Installed {
		res = ExtractVersion(re, stderr.String())
	}
	res.RawOutput = stdout.String() + stderr.String()
	// The binary is on PATH, so it counts as installed even when the version
	// could not be parsed; surface the problem via Err instead.
	if !res.Installed {
		res.Installed = true
		if runErr != nil {
			res.Err = runErr
		} else {
			res.Err = errors.New("detect: version pattern not found in command output")
		}
	}
	return res, nil
}

// ExtractVersion applies the compiled detect regex to command output.
// Split out for direct unit testing against fixture outputs.
func ExtractVersion(re *regexp.Regexp, output string) model.DetectResult {
	m := re.FindStringSubmatch(output)
	if len(m) < 2 {
		return model.DetectResult{Installed: false}
	}
	return model.DetectResult{Installed: true, Version: m[1]}
}
