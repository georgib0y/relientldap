package main

import (
	"log"
	"net"
	"os"

	"github.com/georgib0y/relientldap/internal/app"
	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/internal/server"
)

var logger = log.New(os.Stderr, "main: ", log.Lshortfile)

type Config struct {
	attributeLdifPath   string
	objectClassLdifPath string
}

func loadSchema(config Config) (*d.Schema, error) {
	fattr, err := os.Open(config.attributeLdifPath)
	if err != nil {
		return nil, err
	}
	defer fattr.Close()

	focs, err := os.Open(config.objectClassLdifPath)
	if err != nil {
		return nil, err
	}
	defer focs.Close()

	return d.LoadSchemaFromReaders(fattr, focs)

}

func main() {
	// TODO remove hardcoded config
	config := Config{
		attributeLdifPath:   "ldif/attributes.ldif",
		objectClassLdifPath: "ldif/objClasses.ldif",
	}

	schema, err := loadSchema(config)
	if err != nil {
		logger.Fatalf("could not load schema: %s", err)
	}

	// TODO load dit from persisted modelb
	dit := d.GenerateTestDIT(schema)

	logger.Print("generated schema and test dit")

	scheduler := app.NewScheduler(dit, schema)
	logger.Print("running scheduler in other goroutine")

	mux := server.NewMux()

	bindService := app.NewBindService(schema, scheduler)
	mux.AddHandler(server.NewBindHandler(bindService))
	mux.AddHandler(server.UnbindHandler)

	addService := app.NewAddService(schema, scheduler)
	mux.AddHandler(server.NewAddHandler(addService))

	modifyService := app.NewModifyService(schema, scheduler)
	mux.AddHandler(server.NewModifyHandler(modifyService))
	mux.AddHandler(server.NewModifyDnHandler(modifyService))

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
