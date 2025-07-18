package server

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/georgib0y/relientldap/pkg/ber"
)

// the real attribute deinition requies that vals is not empty
type Attribute struct {
	aType string
	vals  ber.Set[string]
}

type AddRequest struct {
	entry      string
	attributes []Attribute
}

func HandleAddRequest(ctx context.Context, w io.Writer, s *Scheduler, msg LdapMsg) (context.Context, error) {
	logger.Print("in add request")

	_, req, ok := msg.Request.Chosen()
	if !ok {
		return ctx, fmt.Errorf("could not get choice for add request")
	}

	ar, ok := req.(*AddRequest)
	if !ok {
		return ctx, fmt.Errorf("expected *AddRequest, got %s", reflect.TypeOf(req))
	}

	_ = ar

	return ctx, nil
}
