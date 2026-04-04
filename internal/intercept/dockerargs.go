package intercept

import "fmt"

type DockerArgsPlan struct {
	Operation   Operation `json:"operation" yaml:"operation"`
	ComposePath string    `json:"composePath,omitempty" yaml:"composePath,omitempty"`
	WithVolumes bool      `json:"withVolumes" yaml:"withVolumes"`
	Detached    bool      `json:"detached" yaml:"detached"`
	Build       bool      `json:"build" yaml:"build"`
}

func ParseDockerArgs(args []string) (DockerArgsPlan, error) {
	if len(args) == 0 {
		return DockerArgsPlan{}, fmt.Errorf("docker args are required")
	}
	if args[0] == "compose" {
		return parseComposeArgs(args[1:])
	}
	if len(args) >= 2 && args[0] == "volume" && args[1] == "rm" {
		return DockerArgsPlan{Operation: OpDockerVolumeRemove}, nil
	}
	if len(args) >= 2 && args[0] == "system" && args[1] == "prune" {
		return DockerArgsPlan{Operation: OpDockerSystemPrune}, nil
	}
	return DockerArgsPlan{}, fmt.Errorf("unsupported docker interception args: %v", args)
}

func parseComposeArgs(args []string) (DockerArgsPlan, error) {
	plan := DockerArgsPlan{
		Operation: OpComposeDown,
		Detached:  true,
		Build:     true,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--file":
			if i+1 >= len(args) {
				return DockerArgsPlan{}, fmt.Errorf("compose file flag requires a value")
			}
			plan.ComposePath = args[i+1]
			i++
		case "down":
			plan.Operation = OpComposeDown
		case "-v", "--volumes":
			plan.WithVolumes = true
			plan.Operation = OpComposeDownWithVolumes
		case "up":
			plan.Operation = Operation("compose.up")
		case "-d", "--detach":
			plan.Detached = true
		case "--build":
			plan.Build = true
		}
	}
	if plan.ComposePath == "" && (plan.Operation == OpComposeDown || plan.Operation == OpComposeDownWithVolumes || plan.Operation == Operation("compose.up")) {
		return DockerArgsPlan{}, fmt.Errorf("compose interception requires -f/--file")
	}
	return plan, nil
}
