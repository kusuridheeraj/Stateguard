package intercept

import "fmt"

type DockerArgsPlan struct {
	Operation   Operation `json:"operation" yaml:"operation"`
	ComposePath string    `json:"composePath,omitempty" yaml:"composePath,omitempty"`
	WithVolumes bool      `json:"withVolumes" yaml:"withVolumes"`
	Detached    bool      `json:"detached" yaml:"detached"`
	Build       bool      `json:"build" yaml:"build"`
	Targets     []string  `json:"targets,omitempty" yaml:"targets,omitempty"`
	Flags       []string  `json:"flags,omitempty" yaml:"flags,omitempty"`
}

func ParseDockerArgs(args []string) (DockerArgsPlan, error) {
	if len(args) == 0 {
		return DockerArgsPlan{}, fmt.Errorf("docker args are required")
	}
	if args[0] == "compose" {
		return parseComposeArgs(args[1:])
	}
	if len(args) >= 2 && args[0] == "rm" {
		targets, flags := splitDockerFlagsAndTargets(args[1:])
		withVolumes := containsFlag(flags, "-v") || containsFlag(flags, "--volumes")
		if len(targets) == 0 {
			return DockerArgsPlan{}, fmt.Errorf("docker rm requires at least one target container")
		}
		return DockerArgsPlan{
			Operation:   OpDockerRemove,
			Targets:     targets,
			Flags:       flags,
			WithVolumes: withVolumes,
		}, nil
	}
	if len(args) >= 2 && args[0] == "volume" && args[1] == "rm" {
		targets, flags := splitDockerFlagsAndTargets(args[2:])
		if len(targets) == 0 {
			return DockerArgsPlan{}, fmt.Errorf("docker volume rm requires at least one target volume")
		}
		return DockerArgsPlan{
			Operation: OpDockerVolumeRemove,
			Targets:   targets,
			Flags:     flags,
		}, nil
	}
	if len(args) >= 2 && args[0] == "system" && args[1] == "prune" {
		_, flags := splitDockerFlagsAndTargets(args[2:])
		return DockerArgsPlan{
			Operation: OpDockerSystemPrune,
			Flags:     flags,
		}, nil
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
			plan.Operation = OpComposeUp
		case "-d", "--detach":
			plan.Detached = true
		case "--build":
			plan.Build = true
		}
	}
	if plan.ComposePath == "" && (plan.Operation == OpComposeDown || plan.Operation == OpComposeDownWithVolumes || plan.Operation == OpComposeUp) {
		return DockerArgsPlan{}, fmt.Errorf("compose interception requires -f/--file")
	}
	return plan, nil
}

func splitDockerFlagsAndTargets(args []string) ([]string, []string) {
	targets := make([]string, 0, len(args))
	flags := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 0 && arg[0] == '-' {
			flags = append(flags, arg)
			if consumesNextAsValue(arg) && i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		targets = append(targets, arg)
	}
	return targets, flags
}

func consumesNextAsValue(flag string) bool {
	switch flag {
	case "--filter":
		return true
	default:
		return false
	}
}
