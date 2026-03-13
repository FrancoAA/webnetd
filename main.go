package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	flag "github.com/spf13/pflag"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "listen address (host:port)")
	shell := flag.String("shell", "", "shell to execute (default: user's login shell or /bin/sh)")
	authEnabled := flag.Bool("auth", false, "enable PIN authentication")
	uploadDir := flag.String("upload-dir", ".", "directory for file uploads")
	showVersion := flag.BoolP("version", "v", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("webnetd", version)
		os.Exit(0)
	}

	loginShell := *shell
	if loginShell == "" {
		loginShell = os.Getenv("SHELL")
		if loginShell == "" {
			loginShell = "/bin/sh"
		}
	}

	srv := newServer(loginShell, *authEnabled, *uploadDir)

	log.Printf("webnetd listening on %s (shell: %s, auth: %v, upload-dir: %s)",
		*addr, loginShell, *authEnabled, *uploadDir)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
