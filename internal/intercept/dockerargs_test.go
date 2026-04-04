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

func TestParseDockerSystemPrune(t *testing.T) {
	plan, err := ParseDockerArgs([]string{"system", "prune"})
	if err != nil {
		t.Fatalf("parse docker args: %v", err)
	}
	if plan.Operation != OpDockerSystemPrune {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}
