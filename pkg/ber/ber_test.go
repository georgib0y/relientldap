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
	trueb := bytes.NewBuffer([]byte{0x01, 0x01, 0xFF})

	var berbool BerBool
	err := Decode(trueb, &berbool)
	if err != nil {
		t.Fatal(err)
	}

	if berbool != true {
		t.Fatalf("berbool expected to be true but was false")
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

		{-16, []byte{0x02, 0x01, 0xF0}},        // -16
		{-256, []byte{0x02, 0x02, 0xFF, 0x00}}, // -256
		{-257, []byte{0x02, 0x02, 0xFE, 0xFF}}, // -257
		{-1, []byte{0x02, 0x01, 0xFF}},         // ??
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

func (BindRequestSimple) Class() Class {
	return Application
}

func (BindRequestSimple) Construction() Construction {
	return Constructed
}

func (BindRequestSimple) TagValue() int {
	return 0
}

func (b BindRequestSimple) EncodeContents(w io.Writer) (int64, error) {
	return encodeSequence(w, b)
}

func (b *BindRequestSimple) DecodeContents(r io.Reader, len int64) error {
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

	if !reflect.DeepEqual(buf.Bytes(), exp) {
		t.Fatalf("encoding %s did not match expected %s", bytesAsHex(buf.Bytes()), bytesAsHex(exp))
	}

}
