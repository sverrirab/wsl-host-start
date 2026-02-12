// wstart is the WSL-side CLI that launches Windows programs via ShellExecuteEx.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sverrirab/wsl-host-start/internal/launch"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

var version = "dev"

func main() {
	verb := flag.String("verb", "", "ShellExecuteEx verb: open, runas, edit, print, explore, properties")
	dir := flag.String("dir", "", "Working directory (WSL or Windows path)")
	wait := flag.Bool("wait", false, "Wait for the launched process to exit")
	min := flag.Bool("min", false, "Start window minimized")
	max := flag.Bool("max", false, "Start window maximized")
	hidden := flag.Bool("hidden", false, "Start window hidden")
	dryRun := flag.Bool("dry-run", false, "Print translated command without executing")
	verbose := flag.Bool("verbose", false, "Print diagnostic info")
	refreshDrives := flag.Bool("refresh-drives", false, "Refresh drive cache and exit")
	versionFlag := flag.Bool("version", false, "Print version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wstart [flags] <target> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "Launch Windows programs from WSL via ShellExecuteEx.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  wstart document.pdf            Open in default PDF viewer\n")
		fmt.Fprintf(os.Stderr, "  wstart .                       Open current directory in Explorer\n")
		fmt.Fprintf(os.Stderr, "  wstart https://google.com      Open URL in default browser\n")
		fmt.Fprintf(os.Stderr, "  wstart -verb runas cmd.exe     Launch elevated command prompt\n")
		fmt.Fprintf(os.Stderr, "  wstart -verb print report.docx Print a document\n")
		fmt.Fprintf(os.Stderr, "  wstart -wait installer.exe     Wait for process to exit\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if *refreshDrives {
		if err := launch.RefreshDrives(); err != nil {
			fatal(err)
		}
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	show := protocol.ShowNormal
	switch {
	case *min:
		show = protocol.ShowMin
	case *max:
		show = protocol.ShowMax
	case *hidden:
		show = protocol.ShowHidden
	}

	opts := &launch.Options{
		Target:  flag.Arg(0),
		Args:    flag.Args()[1:],
		Verb:    *verb,
		WorkDir: *dir,
		Show:    show,
		Wait:    *wait,
		DryRun:  *dryRun,
		Verbose: *verbose,
	}

	result, err := launch.Run(opts)
	if err != nil {
		fatal(err)
	}

	os.Exit(result.ExitCode)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "wstart: %v\n", err)
	os.Exit(1)
}
