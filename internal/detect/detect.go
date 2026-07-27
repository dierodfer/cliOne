// Package detect runs a tool's version command and extracts the installed
// version with the catalog regex.
package detect

import (
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

// RunDetect executes the tool's detect command and extracts the version from
// its combined stdout+stderr using the tool's regex.
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

	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	out, runErr := cmd.CombinedOutput()
	raw := string(out)

	res := ExtractVersion(re, raw)
	res.RawOutput = raw
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
