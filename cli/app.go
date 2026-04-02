package cli

import "fmt"

type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

func Execute(args []string, build VersionInfo) error {
	if len(args) == 0 {
		return printRootHelp()
	}

	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "init":
		return runInit(args[1:])
	case "list":
		return runList(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "version":
		return runVersion(build, args[1:])
	case "ui":
		return runUI(args[1:])
	case "-h", "--help", "help":
		return printRootHelp()
	default:
		return runBuild(args)
	}
}

func printRootHelp() error {
	_, err := fmt.Println(`pmp assembles prompts from reusable markdown blocks.

Usage:
  pmp --preset feature -m "Add CSV export"
  pmp -p review -m "Review auth flow"
  pmp build --preset bugfix -m "Fix race condition"

Commands:
  build      Assemble a prompt explicitly
  init       Scaffold .pmp config and starter blocks
  list       Show presets and available blocks
  doctor     Validate configuration and block resolution
  version    Print version

Build flags:
  -p, --preset <name>
  -m, --message <text>
  --block <a,b,c>
  --var <key=value>
  --token-limit <n>
  --dry-run
  --no-copy
  --out <file>
  --json`)
	return err
}
