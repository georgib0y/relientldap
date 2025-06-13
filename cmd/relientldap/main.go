package main

import (
	"log"
	"net"
	"os"

	"github.com/georgib0y/relientldap/internal/conn"
	"github.com/georgib0y/relientldap/internal/ldif"
	m "github.com/georgib0y/relientldap/internal/model"
)

var logger = log.New(os.Stderr, "main: ", log.Lshortfile)

type Config struct {
	attributeLdifPath   string
	objectClassLdifPath string
}

func loadAttrs(path string) (map[m.OID]*m.Attribute, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, stat.Mode())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p := ldif.NewAttributeParser()
	return ldif.ParseReader(f, p)
}

func loadObjClasses(path string, attrs map[m.OID]*m.Attribute) (map[m.OID]*m.ObjectClass, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, stat.Mode())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p := ldif.NewObjectClassParser(attrs)
	return ldif.ParseReader(f, p)
}

func main() {
	config := Config{
		attributeLdifPath:   "ldif/attributes.ldif",
		objectClassLdifPath: "ldif/objClasses.ldif",
	}

	attrs, err := loadAttrs(config.attributeLdifPath)
	if err != nil {
		logger.Fatalf("could not load attributes: %s", err)
	}

	objClasses, err := loadObjClasses(config.objectClassLdifPath, attrs)
	if err != nil {
		logger.Fatalf("could not load object classes: %s", err)
	}

	schema := m.NewSchema(attrs, objClasses)
	dit := m.GenerateTestDIT(schema)

	scheduler := conn.NewDitScheduler(dit, schema)
	go scheduler.Run()

	mux := conn.NewMux(scheduler)
	mux.AddHandler(conn.BindRequestTag, conn.HandleBindRequest)

	l, err := net.Listen("tcp", ":8000")
	if err != nil {
		logger.Fatal(err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			logger.Fatal(err)
		}

		go mux.Serve(c)
	}

}
