// wstart-host is the Windows-side helper binary.
// It is invoked by the WSL-side wstart CLI over stdin/stdout.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sverrirab/wsl-host-start/internal/drives"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
	"github.com/sverrirab/wsl-host-start/internal/shellexec"
)

var version = "dev"

func main() {
	drivesMode := flag.Bool("drives", false, "Enumerate drives and print JSON to stdout")
	launchMode := flag.Bool("launch", false, "Read LaunchRequest from stdin, execute, print LaunchResponse to stdout")
	versionFlag := flag.Bool("version", false, "Print version")
	flag.Parse()

	switch {
	case *versionFlag:
		fmt.Println(version)
	case *drivesMode:
		if err := runDrives(); err != nil {
			fatal(err)
		}
	case *launchMode:
		if err := runLaunch(); err != nil {
			fatal(err)
		}
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func runDrives() error {
	resp, err := drives.Enumerate()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func runLaunch() error {
	var req protocol.LaunchRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return fmt.Errorf("decoding launch request: %w", err)
	}

	resp := shellexec.Execute(&req)

	return json.NewEncoder(os.Stdout).Encode(resp)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
	os.Exit(1)
}
