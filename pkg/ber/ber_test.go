package ber

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func bytesAsHex(b []byte) string {
	var sb strings.Builder

	sb.WriteString("{ ")
	for _, v := range b {
		fmt.Fprintf(&sb, "0x%02x, ", v)
	}
	sb.WriteString("}")

	return sb.String()
}

func TestEncodeBool(t *testing.T) {
	testBool := true

	var buf bytes.Buffer
	_, err := Encode(&buf, &testBool)
	if err != nil {
		t.Fatal(err)
	}

	exp := []byte{0x01, 0x01, 0xFF}
	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding %s did not match expected %s", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}
}

func TestDecodeBool(t *testing.T) {
	trueb := []byte{0x01, 0x01, 0xFF}
	truebuf := bytes.NewBuffer(trueb)
	var trueber bool
	if err := Decode(truebuf, &trueber); err != nil {
		t.Fatal(err)
	}
	if trueber != true {
		t.Fatalf("berbool expected to be true but was false")
	}

	falseb := []byte{0x01, 0x01, 0x00}
	falsebuf := bytes.NewBuffer(falseb)
	var falseber bool
	if err := Decode(falsebuf, &falseber); err != nil {
		t.Fatal(err)
	}
	if falseber != false {
		t.Fatalf("berbool expected to false but was true")
	}

	badb := []byte{0x01, 0x01, 0x12}
	badbuf := bytes.NewBuffer(badb)
	var badber bool
	if err := Decode(badbuf, &badber); err == nil {
		t.Fatalf("expected error when decoding %s", bytesAsHex(badb))
	}
}

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		v   int
		exp []byte
	}{
		{0x0F, []byte{0x02, 0x01, 0x0F}},
		{0xFFF, []byte{0x02, 0x02, 0x0F, 0xFF}},
		{0xFFFFF, []byte{0x02, 0x03, 0xF, 0xFF, 0xFF}},
		{0xFFFFFFF, []byte{0x02, 0x04, 0x0F, 0xFF, 0xFF, 0xFF}},

		{-16, []byte{0x02, 0x01, 0xF0}},
		{-256, []byte{0x02, 0x02, 0xFF, 0x00}},
		{-257, []byte{0x02, 0x02, 0xFE, 0xFF}},
		{-1, []byte{0x02, 0x01, 0xFF}},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		_, err := Encode(&buf, &test.v)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(buf.Bytes(), test.exp) {

			t.Errorf("0x%X (%d): encoding\n\t%s\ndid not match expected\n\t%s", test.v, test.v, bytesAsHex(buf.Bytes()), bytesAsHex(test.exp))
		}
	}
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		b   []byte
		exp int
	}{
		{[]byte{0x02, 0x01, 0x0F}, 0x0F},
		{[]byte{0x02, 0x02, 0x0F, 0xFF}, 0x0FFF},
		{[]byte{0x02, 0x03, 0xF, 0xFF, 0xFF}, 0x0FFFFF},
		{[]byte{0x02, 0x04, 0x0F, 0xFF, 0xFF, 0xFF}, 0x0FFFFFFF},

		{[]byte{0x02, 0x01, 0xF0}, -16},
		{[]byte{0x02, 0x02, 0xFF, 0x00}, -256},
		{[]byte{0x02, 0x02, 0xFE, 0xFF}, -257},
		{[]byte{0x02, 0x01, 0xFF}, -1},
	}

	for _, test := range tests {
		var i int
		buf := bytes.NewBuffer(test.b)
		if err := Decode(buf, &i); err != nil {
			t.Fatal(err)
		}
		if i != test.exp {
			t.Fatalf("decoded %d not equal to expected %d", i, test.exp)
		}
	}
}

var (
	BindRequestTag = Tag{Application, Constructed, 0}
	BrSimpleTag    = Tag{ContextSpecific, Constructed, 0}
	BrSaslTag      = Tag{ContextSpecific, Constructed, 3}
)

type LdapMsg struct {
	MessageId int
	Request   *Choice[LdapMsgChoice]
	Controls  *Optional[Control] `ber:"class=context-specific,cons=constructed,val=0"`
}

type Control struct {
	ControlType  string
	Criticality  bool
	ControlValue string
}

type LdapMsgChoice struct {
	BindRequest BindRequest `ber:"class=context-specific,cons=constructed,val=0"`
}

type BindRequest struct {
	Version int
	Name    string
	Auth    *Choice[BindReqChoice]
}

type BindReqChoice struct {
	Simple string   `ber:"class=context-specific,cons=constructed,val=0"`
	Sasl   SaslAuth `ber:"class=context-specific,cons=constructed,val=3"`
}

type SaslAuth struct {
	Mechanism   string
	Credentials []byte
}

func TestEncodeBindRequestSimple(t *testing.T) {
	auth, err := NewChosen[BindReqChoice](BrSimpleTag, "123")
	if err != nil {
		t.Fatal(err)
	}
	br, err := NewChosen[LdapMsgChoice](
		BindRequestTag,
		BindRequest{Version: 3, Name: "test", Auth: auth},
	)
	if err != nil {
		t.Fatal(err)
	}

	msg := LdapMsg{MessageId: 1, Request: br}

	exp := []byte{
		0x30, 0x13, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x0E, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed, 0x03, 0x31, 0x32, 0x33, // cred: "123"
	}

	var buf bytes.Buffer
	if _, err := Encode(&buf, &msg); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding:\n%s\ndid not match expected:\n%s\n", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}
}

func TestDecodeBindRequestSimple(t *testing.T) {
	var br LdapMsg

	b := []byte{
		0x30, 0x13, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x0E, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed, 0x03, 0x31, 0x32, 0x33, // cred: "123"
	}

	buf := bytes.NewBuffer(b)

	if err := Decode(buf, &br); err != nil {
		t.Fatal(err)
	}

	authexp, err := NewChosen[BindReqChoice](BrSimpleTag, "123")
	if err != nil {
		t.Fatal(err)
	}
	brexp, err := NewChosen[LdapMsgChoice](BindRequestTag, BindRequest{Version: 3, Name: "test", Auth: authexp})
	exp := LdapMsg{MessageId: 1, Request: brexp}

	if br.MessageId != exp.MessageId {
		t.Fatalf("decoded msgid %d not eq to exp %d", br.MessageId, exp.MessageId)
	}

	brReq := br.Request.Choices.BindRequest
	expReq := exp.Request.Choices.BindRequest
	switch {
	case brReq.Version != expReq.Version:
		t.Fatalf("decoded version %d not eq to exp %d", brReq.Version, expReq.Version)
	case !reflect.DeepEqual(brReq.Name, expReq.Name):
		t.Fatalf("decoded name %s not eq to exp %s", string(brReq.Name), string(expReq.Name))
	case !reflect.DeepEqual(brReq.Auth.Choices.Simple, expReq.Auth.Choices.Simple):
		t.Fatalf("decoded simple %s not eq to exp %s", string(brReq.Auth.Choices.Simple), string(expReq.Auth.Choices.Simple))
	}
}

func TestEncodeBindRequestSimpleWithControls(t *testing.T) {
	auth, err := NewChosen[BindReqChoice](BrSimpleTag, "123")
	if err != nil {
		t.Fatal(err)
	}
	br, err := NewChosen[LdapMsgChoice](
		BindRequestTag,
		BindRequest{Version: 3, Name: "test", Auth: auth},
	)
	if err != nil {
		t.Fatal(err)
	}

	msg := LdapMsg{MessageId: 1, Request: br, Controls: NewOptional(Control{
		ControlType:  "type",
		Criticality:  true,
		ControlValue: "val",
	})}

	exp := []byte{
		0x30, 0x23, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x0E, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed, 0x03, 0x31, 0x32, 0x33, // cred: "123"
		ContextSpecific | Constructed, 0x0E, // controls tag/len
		0x04, 0x04, 0x74, 0x79, 0x70, 0x65, // control type "type"
		0x01, 0x01, 0xFF, // criticality "true"
		0x04, 0x03, 0x76, 0x61, 0x6C, // control value "val"
	}

	var buf bytes.Buffer
	if _, err := Encode(&buf, &msg); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding:\n%s\ndid not match expected:\n%s\n", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}
}

func TestDecodeBindRequestSimpleWithControls(t *testing.T) {
	var br LdapMsg

	b := []byte{
		0x30, 0x23, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x0E, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed, 0x03, 0x31, 0x32, 0x33, // cred: "123"
		ContextSpecific | Constructed, 0x0E, // controls tag/len
		0x04, 0x04, 0x74, 0x79, 0x70, 0x65, // control type "type"
		0x01, 0x01, 0xFF, // criticality "true"
		0x04, 0x03, 0x76, 0x61, 0x6C, // control value "val"
	}

	buf := bytes.NewBuffer(b)

	if err := Decode(buf, &br); err != nil {
		t.Fatal(err)
	}

	authexp, err := NewChosen[BindReqChoice](BrSimpleTag, "123")
	if err != nil {
		t.Fatal(err)
	}
	brexp, err := NewChosen[LdapMsgChoice](BindRequestTag, BindRequest{Version: 3, Name: "test", Auth: authexp})
	exp := LdapMsg{MessageId: 1, Request: brexp, Controls: NewOptional(Control{
		ControlType:  "type",
		Criticality:  true,
		ControlValue: "val",
	})}

	if br.MessageId != exp.MessageId {
		t.Fatalf("decoded msgid %d not eq to exp %d", br.MessageId, exp.MessageId)
	}

	brReq := br.Request.Choices.BindRequest
	expReq := exp.Request.Choices.BindRequest
	switch {
	case brReq.Version != expReq.Version:
		t.Fatalf("decoded version %d not eq to exp %d", brReq.Version, expReq.Version)
	case !reflect.DeepEqual(brReq.Name, expReq.Name):
		t.Fatalf("decoded name %s not eq to exp %s", string(brReq.Name), string(expReq.Name))
	case !reflect.DeepEqual(brReq.Auth.Choices.Simple, expReq.Auth.Choices.Simple):
		t.Fatalf("decoded simple %s not eq to exp %s", string(brReq.Auth.Choices.Simple), string(expReq.Auth.Choices.Simple))
	}
}

func TestEncodeBindRequestSasl(t *testing.T) {
	auth, err := NewChosen[BindReqChoice](BrSaslTag, SaslAuth{Mechanism: "m", Credentials: []byte("123")})
	if err != nil {
		t.Fatal(err)
	}
	br, err := NewChosen[LdapMsgChoice](
		BindRequestTag,
		BindRequest{Version: 3, Name: "test", Auth: auth},
	)
	if err != nil {
		t.Fatal(err)
	}

	msg := LdapMsg{MessageId: 1, Request: br}

	exp := []byte{
		0x30, 0x18, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x13, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed | 0x03, 0x08, // sasl tag/len
		0x04, 0x01, 0x6D, // mech "m"
		0x04, 0x03, 0x31, 0x32, 0x33, // creds "123"
	}

	var buf bytes.Buffer
	if _, err = Encode(&buf, &msg); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding:\n%s\ndid not match expected:\n%s\n", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}

}

func TestDecodeBindRequestSasl(t *testing.T) {
	var br LdapMsg

	b := []byte{
		0x30, 0x18, //ldapmsg tag/len
		0x02, 0x01, 0x01, //msgid: 1
		0x60, 0x13, // bind req tag/len
		0x02, 0x01, 0x03, // version: 3
		0x04, 0x04, 0x74, 0x65, 0x73, 0x74, // name: "test"
		ContextSpecific | Constructed | 0x03, 0x08, // sasl tag/len
		0x04, 0x01, 0x6D, // mechanism "m"
		0x04, 0x03, 0x31, 0x32, 0x33, // credentials "123"
	}

	buf := bytes.NewBuffer(b)

	if err := Decode(buf, &br); err != nil {
		t.Fatal(err)
	}

	authexp, err := NewChosen[BindReqChoice](BrSaslTag, SaslAuth{Mechanism: "m", Credentials: []byte("123")})
	if err != nil {
		t.Fatal(err)
	}
	brexp, err := NewChosen[LdapMsgChoice](BindRequestTag, BindRequest{Version: 3, Name: "test", Auth: authexp})
	exp := LdapMsg{MessageId: 1, Request: brexp}

	brReq := br.Request.Choices.BindRequest
	expReq := exp.Request.Choices.BindRequest
	brSasl := brReq.Auth.Choices.Sasl
	expSasl := brReq.Auth.Choices.Sasl

	switch {
	case brReq.Version != expReq.Version:
		t.Fatalf("decoded version %d not eq to exp %d", brReq.Version, expReq.Version)
	case !reflect.DeepEqual(brReq.Name, expReq.Name):
		t.Fatalf("decoded name %s not eq to exp %s", string(brReq.Name), string(expReq.Name))
	case !reflect.DeepEqual(brSasl.Mechanism, expSasl.Mechanism):
		t.Fatalf("decoded mechanism %s not eq to exp %s", brSasl.Mechanism, expSasl.Mechanism)
	case !reflect.DeepEqual(brSasl.Credentials, expSasl.Credentials):
		t.Fatalf("decoded mechanism %s not eq to exp %s", string(brSasl.Credentials), string(expSasl.Credentials))
	}
}
