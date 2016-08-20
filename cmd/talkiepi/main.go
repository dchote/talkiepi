package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/dchote/go-rpio"
	"github.com/dchote/talkiepi"
	"github.com/layeh/barnard/uiterm"
	"github.com/layeh/gumble/gumble"
	_ "github.com/layeh/gumble/opus"
	"os"
)

func main() {
	// Command line flags
	server := flag.String("server", "172.20.0.202:64738", "the server to connect to")
	username := flag.String("username", "", "the username of the client")
	password := flag.String("password", "", "the password of the server")
	insecure := flag.Bool("insecure", true, "skip server certificate verification")
	certificate := flag.String("certificate", "", "PEM encoded certificate and private key")

	flag.Parse()

	// Initialize
	b := talkiepi.Talkiepi{
		Config:  gumble.NewConfig(),
		Address: *server,
	}

	b.GPIO = new(rpio.GPIO)

	b.Config.Username = *username
	b.Config.Password = *password

	if *insecure {
		b.TLSConfig.InsecureSkipVerify = true
	}
	if *certificate != "" {
		cert, err := tls.LoadX509KeyPair(*certificate, *certificate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		b.TLSConfig.Certificates = append(b.TLSConfig.Certificates, cert)
	}

	b.Ui = uiterm.New(&b)
	b.Ui.Run()
}
