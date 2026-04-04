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

	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/daemon"
	"github.com/kusuridheeraj/stateguard/internal/dashboardapi"
	"github.com/kusuridheeraj/stateguard/internal/intercept"
	"github.com/kusuridheeraj/stateguard/internal/kube"
	"github.com/kusuridheeraj/stateguard/internal/policy"
	"github.com/kusuridheeraj/stateguard/internal/service"
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

	server, err := daemon.NewServer(logger, cfg, buildInfo)
	if err != nil {
		return err
	}
	return server.Run(ctx)
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
	case "artifacts":
		return runArtifactsCommand(stdout)
	case "adapters":
		return runAdaptersCommand(stdout)
	case "scheduler":
		return runSchedulerCommand(stdout)
	case "retention":
		return runRetentionCommand(stdout)
	case "compose":
		return runComposeCommand(args[1:], stdout)
	case "kube":
		return runKubeCommand(args[1:], stdout)
	case "protect":
		return runProtectCommand(args[1:], stdout)
	case "guard":
		return runGuardCommand(args[1:], stdout)
	case "intercept":
		return runInterceptCommand(args[1:], stdout)
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

	server, err := dashboardapi.NewServer(logger, cfg, buildInfo)
	if err != nil {
		return err
	}
	return server.Run(ctx)
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
	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	return writeJSON(stdout, cp.Status("stateguard-cli"))
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

func runArtifactsCommand(stdout io.Writer) error {
	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"items": cp.Artifacts()})
}

func runAdaptersCommand(stdout io.Writer) error {
	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"items": cp.Adapters()})
}

func runSchedulerCommand(stdout io.Writer) error {
	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"jobs": cp.SchedulerJobs()})
}

func runRetentionCommand(stdout io.Writer) error {
	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	return writeJSON(stdout, cp.RetentionPreview())
}

func runComposeCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("compose requires a subcommand: inspect, down, or up")
	}

	switch args[0] {
	case "inspect":
		fs := flag.NewFlagSet("compose inspect", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		path := fs.String("f", "", "path to compose file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *path == "" {
			return errors.New("compose inspect requires -f path")
		}

		project, err := compose.Discover(*path)
		if err != nil {
			return err
		}
		return writeJSON(stdout, project)
	case "down":
		return runComposeDown(args[1:], stdout)
	case "up":
		return runComposeUp(args[1:], stdout)
	default:
		return fmt.Errorf("unknown compose subcommand %q", args[0])
	}
}

func runProtectCommand(args []string, stdout io.Writer) error {
	if len(args) < 2 || args[0] != "compose" {
		return errors.New("protect requires the subcommand: compose")
	}

	fs := flag.NewFlagSet("protect compose", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("f", "", "path to compose file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("protect compose requires -f path")
	}

	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	report, err := cp.ProtectCompose(context.Background(), *path)
	if err != nil {
		return err
	}
	return writeJSON(stdout, report)
}

func runKubeCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("kube requires a subcommand: inspect or guard-delete")
	}

	switch args[0] {
	case "inspect":
		fs := flag.NewFlagSet("kube inspect", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		path := fs.String("f", "", "path to kubernetes manifests")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *path == "" {
			return errors.New("kube inspect requires -f path")
		}

		descriptor, err := kube.Discover(*path)
		if err != nil {
			return err
		}
		return writeJSON(stdout, descriptor)
	case "guard-delete":
		fs := flag.NewFlagSet("kube guard-delete", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		path := fs.String("f", "", "path to kubernetes manifests")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *path == "" {
			return errors.New("kube guard-delete requires -f path")
		}

		result, err := kube.GuardDelete(*path)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return fmt.Errorf("unknown kube subcommand %q", args[0])
	}
}

func runGuardCommand(args []string, stdout io.Writer) error {
	if len(args) < 2 || args[0] != "compose" {
		return errors.New("guard requires the subcommand: compose")
	}

	fs := flag.NewFlagSet("guard compose", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("f", "", "path to compose file")
	command := fs.String("command", string(intercept.OpComposeDown), "destructive command to preflight")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("guard compose requires -f path")
	}

	cp, err := loadControlPlane()
	if err != nil {
		return err
	}

	result, err := cp.GuardComposeOperation(context.Background(), *path, intercept.Operation(*command))
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runInterceptCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "compose" {
		return errors.New("intercept requires the subcommand: compose")
	}
	if len(args) < 2 {
		return errors.New("intercept compose requires an operation such as down or up")
	}

	switch args[1] {
	case "down":
		return runComposeDown(args[2:], stdout)
	case "up":
		return runComposeUp(args[2:], stdout)
	default:
		return fmt.Errorf("unknown intercept compose operation %q", args[1])
	}
}

func runComposeDown(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("compose down", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("f", "", "path to compose file")
	execute := fs.Bool("execute", false, "execute docker compose down after guard")
	withVolumes := fs.Bool("with-volumes", false, "treat this as volume-destructive")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("compose down requires -f path")
	}

	cp, err := loadControlPlane()
	if err != nil {
		return err
	}
	result, err := cp.InterceptComposeDown(context.Background(), *path, *withVolumes, *execute)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runComposeUp(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("compose up", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("f", "", "path to compose file")
	execute := fs.Bool("execute", false, "execute docker compose up after command preparation")
	detached := fs.Bool("d", true, "run detached")
	build := fs.Bool("build", true, "build before starting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("compose up requires -f path")
	}

	cp, err := loadControlPlane()
	if err != nil {
		return err
	}

	result, err := cp.InterceptComposeUp(context.Background(), *path, *detached, *build, *execute)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "stateguard commands:")
	_, _ = fmt.Fprintln(w, "  version")
	_, _ = fmt.Fprintln(w, "  config validate [-f path]")
	_, _ = fmt.Fprintln(w, "  config print-defaults")
	_, _ = fmt.Fprintln(w, "  status")
	_, _ = fmt.Fprintln(w, "  policy check")
	_, _ = fmt.Fprintln(w, "  artifacts")
	_, _ = fmt.Fprintln(w, "  adapters")
	_, _ = fmt.Fprintln(w, "  scheduler")
	_, _ = fmt.Fprintln(w, "  retention")
	_, _ = fmt.Fprintln(w, "  compose inspect -f compose.yaml")
	_, _ = fmt.Fprintln(w, "  compose down -f compose.yaml [--execute]")
	_, _ = fmt.Fprintln(w, "  compose up -f compose.yaml [--execute]")
	_, _ = fmt.Fprintln(w, "  kube inspect -f manifests.yaml")
	_, _ = fmt.Fprintln(w, "  kube guard-delete -f manifests.yaml")
	_, _ = fmt.Fprintln(w, "  protect compose -f compose.yaml")
	_, _ = fmt.Fprintln(w, "  guard compose -f compose.yaml --command compose.down")
	_, _ = fmt.Fprintln(w, "  intercept compose down -f compose.yaml [--execute]")
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func loadControlPlane() (*service.ControlPlane, error) {
	cfg, err := config.Load(os.Getenv("STATEGUARD_CONFIG"))
	if err != nil {
		return nil, err
	}

	logger := logging.New(logging.Config{
		Level:  slog.LevelInfo,
		Format: "text",
	})

	return service.NewControlPlane(logger, cfg, buildInfo)
}
