package ber

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var logger = log.New(os.Stderr, "ber: ", log.Lshortfile)

const (
	BoolUniversalTag        int = 0x01
	IntUniversalTag             = 0x02
	OctetStringUniversalTag     = 0x04
	NullUniversalTag            = 0x05
	EnumeratedUniversalTag      = 0x05
	SequenceUniversalTag        = 0x30
	SetUniversalTag             = 0x31
)

type Class byte

const (
	Universal       Class = 0x00
	Application           = 0x40
	ContextSpecific       = 0x80
	Private               = 0xC0
)

func classFromStructTag(s string) (Class, error) {
	switch s {
	case "universal":
		return Universal, nil
	case "application":
		return Application, nil
	case "context-specific":
		return ContextSpecific, nil
	case "private":
		return Private, nil
	default:
		return Private, fmt.Errorf("unknown class, struct tag is %s", s)
	}
}

type Construction byte

const (
	Primitive   Construction = 0x00
	Constructed              = 0x20
)

type Tag struct {
	class     Class
	construct Construction
	value     int
}

type BerValue interface {
	Class() Class
	Construction() Construction
	TagValue() int
	EncodeContents(w io.Writer) (int64, error)
	DecodeContents(r io.Reader, len int64) error
}

type BerStructTag struct {
	tag   *int
	class *Class
}

func NewBerStructTag(s string) (BerStructTag, error) {
	kv := map[string]string{}
	for _, s1 := range strings.Split(s, ",") {
		spl := strings.Split(s1, "=")
		if len(spl) != 2 {
			return BerStructTag{}, fmt.Errorf("invalid tag value %q", s1)
		}

		kv[spl[0]] = spl[1]
	}

	var bst BerStructTag

	if tagStr, ok := kv["tag"]; ok {
		tag, err := strconv.Atoi(tagStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid tag value %q: %w", tagStr, err)
		}

		*(bst.tag) = tag
	}

	if classStr, ok := kv["tag"]; ok {
		class, err := classFromStructTag(classStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid class value %q: %w", classStr, err)
		}

		*(bst.class) = class
	}

	return bst, nil
}

type BerBool bool

func (BerBool) Class() Class {
	return Universal
}

func (BerBool) Construction() Construction {
	return Primitive
}

func (BerBool) TagValue() int {
	return BoolUniversalTag
}

func (b BerBool) EncodeContents(w io.Writer) (int64, error) {
	var v byte
	if b {
		v = 0xFF
	}

	n, err := w.Write([]byte{v})
	return int64(n), err
}

func (b *BerBool) DecodeContents(r io.Reader, len int64) error {
	if len != 1 {
		return fmt.Errorf("incorrect byte len (%d) for a boolean, expected 1", len)
	}

	br := bufio.NewReader(r)
	byt, err := br.ReadByte()
	if err != nil {
		return err
	}

	switch byt {
	case 0x00:
		*b = false
	case 0xFF:
		*b = true
	default:
		return fmt.Errorf("unknown byte value 0x%X", byt)
	}

	return nil
}

type BerInt int64

func NewBerInt(i int64) *BerInt {
	b := BerInt(i)
	return &b
}

func (BerInt) Class() Class {
	return Universal
}

func (BerInt) Construction() Construction {
	return Primitive
}

func (BerInt) TagValue() int {
	return IntUniversalTag
}

func reduce(rep64 []byte, neg bool) []byte {
	start := 0

	// if the first byte and the first bit of the second byte are
	// either all 1s or 0s then the first byte is redundant and can be
	// ignored
	for i := 0; i < len(rep64)-1; i++ {
		logger.Printf("%d: 0x%X", i, rep64[i])

		if neg && rep64[i] == 0xFF && rep64[i+1]&0x80 > 0 {
			start += 1
			continue
		}

		if !neg && rep64[i] == 0x00 && rep64[i+1]&0x80 == 0 {
			start += 1
			continue
		}

		break
	}

	return rep64[start:]
}

func encodeInt(w io.Writer, i int64) (int64, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	n, err := w.Write(reduce(b, i < 0))
	return int64(n), err
}

func (i BerInt) EncodeContents(w io.Writer) (int64, error) {
	return encodeInt(w, int64(i))
}

func decodeInt(r io.Reader, len int64) (int64, error) {
	if len > 8 {
		return 0, fmt.Errorf("int len (%d) is too long", len)
	}

	byteRep := make([]byte, 8)
	start := 8 - len
	r.Read(byteRep[start:])

	twos := byte(0x00)
	if byteRep[start]&0x80 > 0 {
		twos = 0xFF
	}
	// padd the start of the int64 with the twos complement that would
	// have been reduced when encoded
	for i := int64(0); i < start; i++ {
		byteRep[i] = twos
	}

	i := int64(binary.BigEndian.Uint64(byteRep))
	return i, nil
}

func (i *BerInt) DecodeContents(r io.Reader, len int64) error {
	decoded, err := decodeInt(r, len)
	if err != nil {
		return err
	}

	*i = BerInt(decoded)
	return nil
}

type BerEnum int

func (BerEnum) Class() Class {
	return Universal
}

func (BerEnum) Construction() Construction {
	return Primitive
}

func (BerEnum) TagValue() int {
	return EnumeratedUniversalTag
}

func (e BerEnum) Encode(w io.Writer) (int64, error) {
	return encodeInt(w, int64(e))
}

func (e *BerEnum) DecodeContents(r io.Reader, len int64) error {
	decoded, err := decodeInt(r, len)
	if err != nil {
		return err
	}

	*e = BerEnum(decoded)
	return nil
}

type BerOctetString []byte

func NewBerOctetString(s []byte) *BerOctetString {
	b := BerOctetString(s)
	return &b
}

func (BerOctetString) Class() Class {
	return Universal
}

func (BerOctetString) Construction() Construction {
	return Primitive
}

func (BerOctetString) TagValue() int {
	return OctetStringUniversalTag
}

func (o BerOctetString) EncodeContents(w io.Writer) (int64, error) {
	n, err := w.Write(o)
	return int64(n), err
}

func (o *BerOctetString) DecodeContents(r io.Reader, len int64) error {
	b := make([]byte, len)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return err
	}

	*o = b
	return nil
}

type BerNull struct{}

func (BerNull) Class() Class {
	return Universal
}

func (BerNull) Construction() Construction {
	return Primitive
}

func (BerNull) TagValue() int {
	return NullUniversalTag
}

func (BerNull) EncodeContents(w io.Writer) (int64, error) {
	return 0, nil
}

// TODO pointer?
func (BerNull) DecodeContents(r io.Reader, len int64) error {
	if len != 0 {
		return fmt.Errorf("null length should be zero, got %d", len)
	}

	return nil
}

// TODO not sure how berset works
// could be more specific with generics
type BerSet map[BerValue]struct{}

func (BerSet) Class() Class {
	return Universal
}

func (BerSet) Construction() Construction {
	return Constructed
}

func (BerSet) TagValue() int {
	return SetUniversalTag
}

func (s BerSet) Encode(w io.Writer) (int, error) {
	written := 0
	for v := range s {
		tag := Tag{class: v.Class(), construct: v.Construction(), value: v.TagValue()}
		n, err := encodeTlv(w, tlv{tag, v})
		written += n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

type tlv struct {
	tag      Tag
	contents BerValue
}

func encodeTag(w io.Writer, t Tag) (int, error) {
	id := byte(t.class) | byte(t.construct)
	if t.value > 30 {
		return 0, fmt.Errorf("multibyte identifiers unsupported")
	}

	id |= byte(t.value)

	return w.Write([]byte{id})
}

func decodeTag(r io.Reader) (Tag, error) {
	// TODO do i really know if bufio reader works like this?
	br := bufio.NewReader(r)
	b, err := br.ReadByte()
	if err != nil {
		return Tag{}, err
	}

	class := Class(b & 0xC0)
	cons := Construction(b & 0x20)
	val := int(b & 0x1F)
	if val == 31 {
		return Tag{}, fmt.Errorf("Multibyte tag values unsupported")
	}

	return Tag{class, cons, val}, nil
}

func encodeLen(w io.Writer, len int64) (int, error) {
	if len < 128 {
		return w.Write([]byte{byte(len)})
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(len))

	start := 0
	for i, byteRep := range buf {
		if byteRep == 0 {
			start = i
		}
	}

	lenlen := byte(8 - start)
	n1, err := w.Write([]byte{0x80 | lenlen})
	if err != nil {
		return n1, err
	}

	n2, err := w.Write(buf[start:])
	return n1 + n2, err
}

func decodeLen(r io.Reader) (int64, error) {
	br := bufio.NewReader(r)
	b, err := br.ReadByte()
	if err != nil {
		return 0, err
	}

	if b < 128 {
		return int64(b), nil
	}

	lenlen := int(b & 0x7F)
	if lenlen > 8 {
		return 0, fmt.Errorf("length is too big to represent in an int64: %d", lenlen)
	}
	buf := make([]byte, 8)
	n, err := br.Read(buf[8-lenlen:])
	if err != nil {
		return 0, err
	} else if n != 8-lenlen {
		return 0, fmt.Errorf("did not read enough bytes, expected %d, read %d", 8-lenlen, n)
	}

	ui := binary.BigEndian.Uint64(buf)
	return int64(ui), nil
}

func tagFromBerStruct(v BerValue, bst BerStructTag) Tag {
	t := Tag{
		class:     v.Class(),
		construct: v.Construction(),
		value:     v.TagValue(),
	}

	if bst.class != nil {
		t.class = *bst.class
	}

	if bst.tag != nil {
		t.value = *bst.tag
	}

	return t
}

func encodeSequence(w io.Writer, seq any) (int64, error) {
	if reflect.TypeOf(seq).Kind() != reflect.Struct {
		return 0, fmt.Errorf("%q is %s and not a struct", reflect.TypeOf(seq).Kind().String(), reflect.TypeOf(seq).Name())
	}

	written := int64(0)

	v := reflect.ValueOf(seq)
	t := reflect.TypeOf(seq)
	for i := range v.NumField() {
		f := v.Field(i)
		b, ok := f.Interface().(BerValue)
		if !ok {
			return written, fmt.Errorf("%s is not an encodable BerValue", f.Type().Name())
		}

		bst, err := NewBerStructTag(t.Field(i).Tag.Get("ber"))
		p := tlv{tag: tagFromBerStruct(b, bst), contents: b}

		n, err := encodeTlv(w, p)
		written += int64(n)
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

func decodeSequence(r io.Reader, len int64, seq any) error {
	v := reflect.ValueOf(seq)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("sequence is not a pointer")
	}

	elm := v.Elem()
	if elm.Kind() != reflect.Struct {
		return fmt.Errorf("seq does not point to a struct")
	}

	// loop through all the fields of the struct, setting the zero
	// value and then calling Decode for the field
	for i := range elm.Type().NumField() {
		f := elm.Field(i)
		f.SetZero()
		b, ok := f.Interface().(BerValue)
		if !ok {
			return fmt.Errorf("field %q is not a BerValue", f.Type().Name())
		}

		if err := Decode(r, b); err != nil {
			// TODO maybe wrap?
			return err
		}
	}

	return nil
}

func encodeTlv(w io.Writer, v tlv) (int, error) {
	written := 0
	n, err := encodeTag(w, v.tag)
	written += n
	if err != nil {
		return written, err
	}

	var buf bytes.Buffer
	contentsLen, err := v.contents.EncodeContents(&buf)
	if err != nil {
		return written, err
	}

	n, err = encodeLen(w, contentsLen)
	written += n
	if err != nil {
		return written, err
	}

	contentsWritten, err := io.Copy(w, &buf)
	written += int(contentsWritten)

	return written, err
}

func Encode(w io.Writer, b BerValue) (int, error) {
	p := tlv{
		tag:      Tag{class: b.Class(), construct: b.Construction(), value: b.TagValue()},
		contents: b,
	}
	return encodeTlv(w, p)
}

func Decode(r io.Reader, b BerValue) error {
	t, err := decodeTag(r)
	if err != nil {
		return err
	}

	if t.class != b.Class() {
		return fmt.Errorf("tag class 0x%X does not match expected 0x%X", t.class, b.Class())
	}

	if t.construct != b.Construction() {
		return fmt.Errorf("tag construction 0x%X does not match expected 0x%X", t.construct, b.Construction())
	}

	if t.value != b.TagValue() {
		return fmt.Errorf("tag value %d does not match expected %d", t.value, b.TagValue())
	}

	len, err := decodeLen(r)
	if err != nil {
		return err
	}

	return b.DecodeContents(r, len)
}
