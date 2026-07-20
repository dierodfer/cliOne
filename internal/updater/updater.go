// Package updater decides and executes the action behind enter/u on a tool
// row. It never installs anything from scratch: the only actions are running a
// tool's own native updater, running its owning manager's upgrade command, or
// opening the official install page.
package updater

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
)

// ExecTimeout bounds update commands so a hung updater never wedges the TUI.
const ExecTimeout = 5 * time.Minute

// Decide picks the UpdateAction for a tool from its status, its source kind,
// and whether the catalog declares a bespoke native updater.
//
// Routing table:
//   - not installed / no updater  -> open the official page
//   - up to date                  -> nothing to do
//   - update available:
//     bespoke UpdateSpec        -> run the native update command
//     manager-owned binary      -> run the manager's synthesized upgrade
//     otherwise                 -> open the official page
func Decide(def model.ToolDef, src model.SourceResult, status model.StatusState, reg registry.Registry) model.UpdateAction {
	switch status {
	case model.StatusNotInstalled, model.StatusNoUpdater:
		return model.ActionOpenOfficialPage
	case model.StatusUpToDate:
		return model.ActionNone
	case model.StatusUpdateAvail:
		if def.Update != nil {
			return model.ActionRunNativeUpdate
		}
		if m, ok := reg.ForKind(src.Kind); ok && len(m.UpdateCommand(def.ID)) > 0 {
			return model.ActionRunManagerUpdate
		}
		return model.ActionOpenOfficialPage
	default:
		return model.ActionNone
	}
}

// Command returns the argv the update action would run, without running it.
// Nil means the action runs no command.
func Command(def model.ToolDef, src model.SourceResult, action model.UpdateAction, reg registry.Registry) []string {
	switch action {
	case model.ActionRunNativeUpdate:
		if def.Update == nil {
			return nil
		}
		return strings.Fields(def.Update.Cmd)
	case model.ActionRunManagerUpdate:
		if m, ok := reg.ForKind(src.Kind); ok {
			return m.UpdateCommand(def.ID)
		}
		return nil
	default:
		return nil
	}
}

// Execute runs the update for a tool whose status is update-available,
// choosing native vs manager command via Decide. It captures stdout and
// stderr separately and never returns a hard error for a failing command;
// failure is expressed in the UpdateResult.
func Execute(ctx context.Context, def model.ToolDef, src model.SourceResult) model.UpdateResult {
	reg := registry.Default()
	action := Decide(def, src, model.StatusUpdateAvail, reg)
	argv := Command(def, src, action, reg)
	if action == model.ActionOpenOfficialPage || len(argv) == 0 {
		return model.UpdateResult{
			Action: action,
			Err:    errors.New("updater: no runnable update command for " + def.ID),
		}
	}

	ctx, cancel := context.WithTimeout(ctx, ExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	res := model.UpdateResult{
		Action: action,
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		res.Success = true
		return res
	}
	res.Err = err
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
	} else {
		res.ExitCode = -1
	}
	return res
}
