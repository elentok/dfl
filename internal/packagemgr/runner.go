package packagemgr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type InstallOptions struct {
	Packages []string
	Tap      string
	Cask     bool
}

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
	Exec   Executor
}

type Executor interface {
	Output(name string, args ...string) ([]byte, error)
	Run(stdout, stderr io.Writer, name string, args ...string) error
}

type OSExecutor struct{}

func (OSExecutor) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (OSExecutor) Run(stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (r Runner) Install(ctx runtimectx.Context, manager string, opts InstallOptions) (int, error) {
	if len(opts.Packages) == 0 {
		return 2, errors.New("install requires at least one package")
	}

	err := ui.Step(r.stdout(), installStepMessage(manager, opts), func() (runtimectx.ResultStatus, string, error) {
		missing, err := r.findMissing(manager, opts)
		if err != nil {
			return "", "", err
		}

		if len(missing) == 0 {
			return runtimectx.StatusSkipped, "already installed", nil
		}

		if ctx.DryRun {
			return runtimectx.StatusSuccess, dryRunDetail(manager, missing, opts), nil
		}

		if manager == "brew" && opts.Tap != "" {
			if err := r.ensureBrewTap(opts.Tap); err != nil {
				return "", "", err
			}
		}

		if err := r.installMissing(manager, missing, opts); err != nil {
			return "", "", err
		}

		return runtimectx.StatusSuccess, installedDetail(manager, missing, opts), nil
	})
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func installStepMessage(manager string, opts InstallOptions) string {
	parts := []string{fmt.Sprintf("Installing %s packages", manager)}
	if opts.Tap != "" && manager == "brew" {
		parts = append(parts, fmt.Sprintf("via tap %s", opts.Tap))
	}
	if opts.Cask && manager == "brew" {
		parts = append(parts, "(casks)")
	}
	if len(opts.Packages) > 0 {
		parts = append(parts, strings.Join(opts.Packages, " "))
	}
	return strings.Join(parts, " ")
}

func dryRunDetail(manager string, missing []string, opts InstallOptions) string {
	parts := []string{}
	if opts.Tap != "" && manager == "brew" {
		parts = append(parts, fmt.Sprintf("would ensure tap %s", opts.Tap))
	}
	parts = append(parts, fmt.Sprintf("would install %s packages: %s", manager, strings.Join(missing, " ")))
	return strings.Join(parts, "; ")
}

func installedDetail(manager string, missing []string, opts InstallOptions) string {
	parts := []string{}
	if opts.Tap != "" && manager == "brew" {
		parts = append(parts, fmt.Sprintf("ensured tap %s", opts.Tap))
	}
	parts = append(parts, fmt.Sprintf("installed %s packages: %s", manager, strings.Join(missing, " ")))
	return strings.Join(parts, "; ")
}

func (r Runner) findMissing(manager string, opts InstallOptions) ([]string, error) {
	switch manager {
	case "brew":
		return r.findMissingBrew(opts)
	case "apt":
		return r.findMissingAPT(opts.Packages)
	case "npm":
		return r.findMissingNPM(opts.Packages)
	case "pipx":
		return r.findMissingPipx(opts.Packages)
	case "cargo":
		return r.findMissingCargo(opts.Packages)
	case "snap":
		return r.findMissingSnap(opts.Packages)
	default:
		return nil, fmt.Errorf("unsupported package manager %q", manager)
	}
}

func (r Runner) installMissing(manager string, missing []string, opts InstallOptions) error {
	switch manager {
	case "brew":
		return r.installBrewPkgs(missing, opts)
	case "apt":
		return r.installAPTPkgs(missing)
	case "npm":
		return r.installNPMPkgs(missing)
	case "pipx":
		return r.installPipxPkgs(missing)
	case "cargo":
		return r.installCargoPkgs(missing)
	case "snap":
		return r.installSnapPkgs(missing)
	default:
		return fmt.Errorf("unsupported package manager %q", manager)
	}
}

func splitLines(output []byte) []string {
	raw := strings.Split(strings.TrimSpace(string(output)), "\n")
	var lines []string
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func (r Runner) exec() Executor {
	if r.Exec != nil {
		return r.Exec
	}
	return OSExecutor{}
}

func (r Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return io.Discard
}

func (r Runner) stderr() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return io.Discard
}

type FakeExecutor struct {
	Outputs map[string][]byte
	Runs    []RunCall
}

type RunCall struct {
	Name string
	Args []string
}

func (f *FakeExecutor) Output(name string, args ...string) ([]byte, error) {
	key := commandKey(name, args...)
	if output, ok := f.Outputs[key]; ok {
		return output, nil
	}
	return nil, fmt.Errorf("unexpected output command: %s", key)
}

func (f *FakeExecutor) Run(stdout, stderr io.Writer, name string, args ...string) error {
	f.Runs = append(f.Runs, RunCall{Name: name, Args: append([]string(nil), args...)})
	return nil
}

func commandKey(name string, args ...string) string {
	var buf bytes.Buffer
	buf.WriteString(name)
	for _, arg := range args {
		buf.WriteByte(' ')
		buf.WriteString(arg)
	}
	return buf.String()
}
