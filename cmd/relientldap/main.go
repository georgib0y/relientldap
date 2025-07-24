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

func loadAttrs(path string) (map[d.OID]*d.Attribute, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, stat.Mode())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	attrs, err := ldif.ParseAttributes(f)
	if err != nil {
		return nil, err
	}

	for _, attr := range attrs {
		logger.Print("\n", attr, "\n")
	}

	return attrs, nil
}

func loadObjClasses(path string, attrs map[d.OID]*d.Attribute) (map[d.OID]*d.ObjectClass, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, stat.Mode())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ldif.ParseObjectClasses(f, attrs)
}

func main() {
	// TODO remove hardcoded config
	config := Config{
		attributeLdifPath:   "ldif/attributes.ldif",
		objectClassLdifPath: "ldif/objClasses.ldif",
	}

	attrs, err := loadAttrs(config.attributeLdifPath)
	if err != nil {
		logger.Fatalf("could not load attributes: %s", err)
	}

	logger.Print("loaded attrs")

	objClasses, err := loadObjClasses(config.objectClassLdifPath, attrs)
	if err != nil {
		logger.Fatalf("could not load object classes: %s", err)
	}

	logger.Print("loaded object classes")

	schema := d.NewSchema(attrs, objClasses)
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
