package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"github.com/layeh/barnard"
	"github.com/layeh/barnard/uiterm"
	"github.com/layeh/gumble/gumble"
	"github.com/layeh/gumble/gumbleutil"
	"github.com/layeh/gumble/gumble_openal"
)

func main() {
	// Command line flags
	server := flag.String("server", "localhost:64738", "the server to connect to")
	username := flag.String("username", "", "the username of the client")
	insecure := flag.Bool("insecure", false, "skip server certificate verification")
	certificate := flag.String("certificate", "", "PEM encoded certificate and private key")

	flag.Parse()

	// Initialize
	b := barnard.Barnard{}
	b.Ui = uiterm.New(&b)

	// Gumble
	b.Config = gumble.NewConfig()
	b.Config.Username = *username
	b.Config.Address = *server
	if *insecure {
		b.Config.TLSConfig.InsecureSkipVerify = true
	}
	if *certificate != "" {
		if cert, err := tls.LoadX509KeyPair(*certificate, *certificate); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		} else {
			b.Config.TLSConfig.Certificates = []tls.Certificate{cert}
		}
	}

	b.Client = gumble.NewClient(b.Config)
	b.Client.Attach(gumbleutil.AutoBitrate)
	b.Client.Attach(&b)
	// Audio
	if os.Getenv("ALSOFT_LOGLEVEL") == "" {
		os.Setenv("ALSOFT_LOGLEVEL", "0")
	}
	if stream, err := gumble_openal.New(b.Client); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	} else {
		b.Stream = stream
	}

	if err := b.Client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	b.Ui.Run()
}
