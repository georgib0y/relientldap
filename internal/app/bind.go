package app

import (
	"log"
	"os"

	d "github.com/georgib0y/relientldap/internal/domain"
)

var bindLogger = log.New(os.Stderr, "bindService: ", log.Lshortfile)

type BindService struct {
	schema    *d.Schema
	scheduler *Scheduler
}

func NewBindService(schema *d.Schema, scheduler *Scheduler) *BindService {
	bindLogger.Print("creating new bind service")
	return &BindService{schema, scheduler}
}

type BindRequest interface {
	Dn() string
	Version() int
	Simple() (string, bool)
	SaslMechanism() (string, bool)
	SaslCredentials() (string, bool)
}

func (b *BindService) Bind(br BindRequest) (*d.Entry, error) {
	if br.Version() != 3 {
		return nil, d.NewLdapError(
			d.ProtocolError,
			"",
			"expected bind request to be version 3, not %d", br.Version,
		)
	}

	if simple, ok := br.Simple(); ok {
		return b.authenticateSimple(br.Dn(), simple)
	}

	return nil, d.NewLdapError(d.AuthMethodNotSupported, "", "sasl or unknown method not supported")
}

func (b *BindService) authenticateSimple(entryDn string, simple string) (*d.Entry, error) {
	bindLogger.Print("in auth simple")

	dn, err := d.NormaliseDN(b.schema, entryDn)
	if err != nil {
		bindLogger.Print(err)
		return nil, err
	}

	bindLogger.Printf("silly me logging your password in plain text: %s", simple)

	return ScheduleAwait(b.scheduler, func(dit d.DIT) (*d.Entry, error) {
		return dit.GetEntry(dn)
	})
}
