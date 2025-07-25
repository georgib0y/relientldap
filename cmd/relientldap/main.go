package main

import (
	"log"
	"net"
	"os"

	"github.com/georgib0y/relientldap/internal/app"
	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/internal/ldif"
	"github.com/georgib0y/relientldap/internal/server"
)

var logger = log.New(os.Stderr, "main: ", log.Lshortfile)

type Config struct {
	attributeLdifPath   string
	objectClassLdifPath string
}

func main() {
	// TODO remove hardcoded config
	config := Config{
		attributeLdifPath:   "ldif/attributes.ldif",
		objectClassLdifPath: "ldif/objClasses.ldif",
	}

	schema, err := ldif.LoadSchmeaFromPaths(config.attributeLdifPath, config.objectClassLdifPath)
	if err != nil {
		logger.Fatalf("could not load schema: %s", err)
	}

	// TODO load dit from persisted modelb
	dit := d.GenerateTestDIT(schema)

	logger.Print("generated schema and test dit")

	scheduler := app.NewScheduler(dit, schema)
	go scheduler.Run()

	logger.Print("running scheduler in other goroutine")

	mux := server.NewMux()

	bindService := app.NewBindService(schema, scheduler)
	bindHandler := server.NewBindHandler(bindService)
	mux.AddHandler(bindHandler)
	mux.AddHandler(server.UnbindHandler)

	addService := app.NewAddService(schema, scheduler)
	addHandler := server.NewAddHandler(addService)
	mux.AddHandler(addHandler)

	logger.Print("added handlers to mux")

	l, err := net.Listen("tcp", ":8000")
	if err != nil {
		logger.Fatal(err)
	}

	logger.Print("created new listner, listening...")
	for {
		c, err := l.Accept()
		if err != nil {
			logger.Fatal(err)
		}

		go mux.Serve(c)
		logger.Print("accepted connection, serving...")
	}

}
