package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "listen address (host:port)")
	shell := flag.String("shell", "", "shell to execute (default: user's login shell or /bin/sh)")
	showVersion := flag.Bool("version", false, "print version and exit")
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

	srv := newServer(loginShell)

	log.Printf("webnetd listening on %s (shell: %s)", *addr, loginShell)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
