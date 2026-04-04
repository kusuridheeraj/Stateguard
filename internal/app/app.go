package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/daemon"
	"github.com/kusuridheeraj/stateguard/internal/dashboardapi"
	"github.com/kusuridheeraj/stateguard/internal/policy"
	"github.com/kusuridheeraj/stateguard/pkg/logging"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

var buildInfo = types.BuildInfo{
	Name:    "stateguard",
	Version: "0.1.0-dev",
	Commit:  "unknown",
	Date:    "unknown",
}

func RunDaemon() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Getenv("STATEGUARD_CONFIG"))
	if err != nil {
		return err
	}

	logger := logging.New(logging.Config{
		Level:  slog.LevelInfo,
		Format: "text",
	})

	return daemon.NewServer(logger, cfg, buildInfo).Run(ctx)
}

func RunCLI(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "version":
		return writeJSON(stdout, buildInfo)
	case "config":
		return runConfigCommand(args[1:], stdout)
	case "status":
		return runStatusCommand(stdout)
	case "policy":
		return runPolicyCommand(args[1:], stdout)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func RunDashboardAPI() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Getenv("STATEGUARD_CONFIG"))
	if err != nil {
		return err
	}

	logger := logging.New(logging.Config{
		Level:  slog.LevelInfo,
		Format: "text",
	})

	return dashboardapi.NewServer(logger, cfg, buildInfo).Run(ctx)
}

func runConfigCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("config requires a subcommand: validate or print-defaults")
	}

	switch args[0] {
	case "validate":
		fs := flag.NewFlagSet("config validate", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		path := fs.String("f", "", "path to safedata.yaml")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := config.Load(*path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "config valid: source=%s mode=%s artifact_path=%s\n", cfg.Source, cfg.Policy.Mode, cfg.Storage.Local.Path)
		return err
	case "print-defaults":
		content, err := config.RenderDefaultYAML()
		if err != nil {
			return err
		}
		_, err = stdout.Write(content)
		return err
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func runStatusCommand(stdout io.Writer) error {
	status := types.SystemStatus{
		ServiceName:     "stateguard-cli",
		Version:         buildInfo.Version,
		Mode:            "bootstrap",
		ConfigSource:    os.Getenv("STATEGUARD_CONFIG"),
		RuntimeTargets:  []types.RuntimeTarget{types.RuntimeCompose, types.RuntimeKubernetes},
		ProtectedScopes: 0,
	}
	return writeJSON(stdout, status)
}

func runPolicyCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("policy requires the subcommand: check")
	}

	cfg, err := config.Load(os.Getenv("STATEGUARD_CONFIG"))
	if err != nil {
		return err
	}

	decision := policy.NewEngine(cfg.Policy).Evaluate(types.ProtectionState{
		RecoveryPointExists: true,
		IntegrityValidated:  true,
		RestoreTested:       true,
	})
	return writeJSON(stdout, decision)
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "stateguard commands:")
	_, _ = fmt.Fprintln(w, "  version")
	_, _ = fmt.Fprintln(w, "  config validate [-f path]")
	_, _ = fmt.Fprintln(w, "  config print-defaults")
	_, _ = fmt.Fprintln(w, "  status")
	_, _ = fmt.Fprintln(w, "  policy check")
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
