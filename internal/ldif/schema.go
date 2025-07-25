package ldif

import (
	"fmt"
	"io"
	"os"

	d "github.com/georgib0y/relientldap/internal/domain"
)

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

	attrs, err := ParseAttributes(f)
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

	return ParseObjectClasses(f, attrs)
}

func LoadSchmeaFromPaths(aPath, ocPath string) (*d.Schema, error) {
	aStat, err := os.Stat(aPath)
	if err != nil {
		return nil, err
	}

	aFile, err := os.OpenFile(aPath, os.O_RDONLY, aStat.Mode())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := aFile.Close(); err != nil {
			logger.Fatalf("could not close %q: %s", aPath, err)
		}
	}()

	ocStat, err := os.Stat(ocPath)
	if err != nil {
		return nil, err
	}

	ocFile, err := os.OpenFile(ocPath, os.O_RDONLY, ocStat.Mode())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := ocFile.Close(); err != nil {
			logger.Fatalf("could not close %q: %s", ocPath, err)
		}
	}()

	return LoadSchemaFromReaders(aFile, ocFile)
}

func LoadSchemaFromReaders(aReader, ocReader io.Reader) (*d.Schema, error) {
	attrs, err := ParseAttributes(aReader)
	if err != nil {
		return nil, fmt.Errorf("could not load attributes: %w", err)
	}
	logger.Print("loaded attrs")

	ocs, err := ParseObjectClasses(ocReader, attrs)
	if err != nil {
		return nil, fmt.Errorf("could not load object classes: %w", err)
	}

	logger.Print("loaded object classes")

	return d.NewSchema(attrs, ocs), nil
}
