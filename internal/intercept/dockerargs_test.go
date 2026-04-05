package intercept

import "testing"

func TestParseDockerComposeDownWithVolumes(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"compose", "-f", "compose.yaml", "down", "-v"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpComposeDownWithVolumes || plan.ComposePath != "compose.yaml" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestParseDockerComposeUp(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"compose", "--file", "compose.yaml", "up", "-d", "--build"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpComposeUp || !plan.Detached || !plan.Build {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestParseDockerVolumeRemoveTargetsAndFlags(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"volume", "rm", "-f", "cache-a", "cache-b"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpDockerVolumeRemove {
		t.Fatalf("unexpected operation: %#v", plan)
	}
	if len(plan.Targets) != 2 || plan.Targets[0] != "cache-a" || plan.Targets[1] != "cache-b" {
		t.Fatalf("unexpected targets: %#v", plan.Targets)
	}
	if len(plan.Flags) != 1 || plan.Flags[0] != "-f" {
		t.Fatalf("unexpected flags: %#v", plan.Flags)
	}
}

func TestParseDockerSystemPrune(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"system", "prune", "--volumes", "-a", "--filter", "label=stateguard", "-f"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpDockerSystemPrune {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if len(plan.Flags) != 5 {
		t.Fatalf("unexpected flags: %#v", plan.Flags)
	}
}

func TestParseDockerRemove(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"rm", "-v", "c1", "c2"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpDockerRemove {
		t.Fatalf("unexpected operation: %#v", plan)
	}
	if !plan.WithVolumes {
		t.Fatalf("expected with-volumes to be true")
	}
	if len(plan.Targets) != 2 || plan.Targets[0] != "c1" || plan.Targets[1] != "c2" {
		t.Fatalf("unexpected targets: %#v", plan.Targets)
	}
}

func TestParseDockerVolumeRemoveRequiresTargets(t *testing.T) {
	if _, err := ParseDockerArgs([]string{"volume", "rm", "-f"}); err == nil {
		t.Fatal("expected error for missing targets")
	}
}
