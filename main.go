package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	install    = flag.Bool("install", false, "")
	uninstall  = flag.Bool("uninstall", false, "")
	dryrun     = flag.Bool("dryrun", false, "Print what would be done without doing it")
	configFile = flag.String("config", "/etc/ingress/config", "path to the ingress configuration file")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	if *install {
		fmt.Fprintf(os.Stderr, "Installing and running ingress manager\n")
		Install()
		return
	}
	if *uninstall {
		fmt.Fprintf(os.Stderr, "Stopping and uninstalling ingress manager\n")
		Uninstall()
		return
	}

	conf, _ := os.ReadFile(*configFile)
	p, err := NewProxy(string(conf))
	if err != nil {
		log.Fatal(err)
	}

	if *dryrun {
		log.Print(p)
		return
	}

	log.Println("Starting ingress proxy")
	log.Fatal(p.Run())
}
