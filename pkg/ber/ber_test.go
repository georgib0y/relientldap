package ber

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
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
	testBool := BerBool(true)

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
	var trueber BerBool
	if err := Decode(truebuf, &trueber); err != nil {
		t.Fatal(err)
	}
	if trueber != true {
		t.Fatalf("berbool expected to be true but was false")
	}

	falseb := []byte{0x01, 0x01, 0x00}
	falsebuf := bytes.NewBuffer(falseb)
	var falseber BerBool
	if err := Decode(falsebuf, &falseber); err != nil {
		t.Fatal(err)
	}
	if falseber != false {
		t.Fatalf("berbool expected to false but was true")
	}

	badb := []byte{0x01, 0x01, 0x12}
	badbuf := bytes.NewBuffer(badb)
	var badber BerBool
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
		t.Log(strconv.FormatInt(int64(test.v), 2))
		t.Log(strconv.FormatInt(int64(test.v), 16))
		bi := BerInt(test.v)
		var buf bytes.Buffer
		_, err := Encode(&buf, &bi)
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
		exp int64
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
		var bi BerInt
		buf := bytes.NewBuffer(test.b)
		if err := Decode(buf, &bi); err != nil {
			t.Fatal(err)
		}
		if int64(bi) != test.exp {
			t.Fatalf("decoded %d not equal to expected %d", bi, test.exp)
		}
	}
}

type BindRequestSimple struct {
	Version *BerInt
	Name    *BerOctetString
	Simple  *BerOctetString
}

func NewBindRequestSimple(version int, name, simple string) BindRequestSimple {
	return BindRequestSimple{
		Version: NewBerInt(int64(version)),
		Name:    NewBerOctetString([]byte(name)),
		Simple:  NewBerOctetString([]byte(simple)),
	}
}

func (*BindRequestSimple) Class() Class {
	return Application
}

func (*BindRequestSimple) Construction() Construction {
	return Constructed
}

func (*BindRequestSimple) TagValue() int {
	return 0
}

func (b *BindRequestSimple) EncodeContents(w io.Writer) (int64, error) {
	return encodeSequence(w, b)
}

func (b *BindRequestSimple) DecodeContents(r io.Reader, len int64) error {
	return decodeSequence(r, len, b)
}

type SaslCreds struct {
	Mechanism   *BerOctetString
	Credentials *BerOctetString
}

func (*SaslCreds) Class() Class {
	return Application
}

func (*SaslCreds) Construction() Construction {
	return Constructed
}

func (*SaslCreds) TagValue() int {
	return 0
}

func (c *SaslCreds) EncodeContents(w io.Writer) (int64, error) {
	return encodeSequence(w, c)
}

func (c *SaslCreds) DecodeContents(r io.Reader, len int64) error {
	return decodeSequence(r, len, c)
}

type BindRequestSasl struct {
	Version *BerInt
	Name    *BerOctetString
	Sasl    *SaslCreds
}

func NewBindRequestSasl(v int, name, mechanism, creds string) BindRequestSasl {
	return BindRequestSasl{
		Version: NewBerInt(int64(v)),
		Name:    NewBerOctetString([]byte(name)),
		Sasl: &SaslCreds{
			Mechanism:   NewBerOctetString([]byte(mechanism)),
			Credentials: NewBerOctetString([]byte(creds)),
		},
	}

}

func (*BindRequestSasl) Class() Class {
	return Application
}

func (*BindRequestSasl) Construction() Construction {
	return Constructed
}

func (*BindRequestSasl) TagValue() int {
	return 0
}

func (b *BindRequestSasl) EncodeContents(w io.Writer) (int64, error) {
	return encodeSequence(w, b)
}

func (b *BindRequestSasl) DecodeContents(r io.Reader, len int64) error {
	return decodeSequence(r, len, b)
}

func TestEncodeBindRequestSimple(t *testing.T) {
	brs := NewBindRequestSimple(3, "test", "123")

	var (
		varEncode    = []byte{0x02, 0x01, 0x03}
		nameEncode   = []byte{0x04, 0x04, 0x74, 0x65, 0x73, 0x74}
		simpleEncode = []byte{0x04, 0x03, 0x31, 0x32, 0x33}
	)

	bTag := byte(Application | Constructed)
	exp := []byte{bTag, 0x0E}
	exp = append(exp, varEncode...)
	exp = append(exp, nameEncode...)
	exp = append(exp, simpleEncode...)

	var buf bytes.Buffer
	_, err := Encode(&buf, &brs)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(bytesAsHex(buf.Bytes()))
	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding %s did not match expected %s", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}

}

func TestDecodeBindRequestSimple(t *testing.T) {
	var br BindRequestSimple

	b := []byte{0x60, 0x0e, 0x02, 0x01, 0x03, 0x04, 0x04, 0x74, 0x65, 0x73, 0x74, 0x04, 0x03, 0x31, 0x32, 0x33}
	buf := bytes.NewBuffer(b)

	if err := Decode(buf, &br); err != nil {
		t.Fatal(err)
	}

	exp := NewBindRequestSimple(3, "test", "123")

	switch {
	case *br.Version != *exp.Version:
		t.Fatalf("decoded version %d not eq to exp %d", *br.Version, *exp.Version)
	case !reflect.DeepEqual(*br.Name, *exp.Name):
		t.Fatalf("decoded name %s not eq to exp %s", string(*br.Name), string(*exp.Name))
	case !reflect.DeepEqual(*br.Simple, *exp.Simple):
		t.Fatalf("decoded simple %s not eq to exp %s", string(*br.Simple), string(*exp.Simple))
	}
}

func TestEncodeBindRequestSasl(t *testing.T) {
	brs := NewBindRequestSasl(3, "test", "m", "123")

	var (
		versEncode = []byte{0x02, 0x01, 0x03}
		nameEncode = []byte{0x04, 0x04, 0x74, 0x65, 0x73, 0x74}
		mechEncode = []byte{0x04, 0x01, 0x6D}
		credEncode = []byte{0x04, 0x03, 0x31, 0x32, 0x33}
	)

	bTag := byte(Application | Constructed)
	exp := []byte{bTag, 0x11}
	exp = append(exp, versEncode...)
	exp = append(exp, nameEncode...)
	exp = append(exp, mechEncode...)
	exp = append(exp, credEncode...)

	var buf bytes.Buffer
	_, err := Encode(&buf, &brs)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(bytesAsHex(buf.Bytes()))
	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding %s did not match expected %s", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}

}
