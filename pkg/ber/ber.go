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
	BoolUniversalTagVal  int = 0x01
	IntUniversalTagVal       = 0x02
	OctetStrUniversalTag     = 0x04
	NullUniversalTagVal      = 0x05
	EnumUniversalTagVal      = 0x0A
	SeqUniversalTagVal       = 0x10
	SetUniversalTagVal       = 0x11
)

type Class byte

const (
	Universal       Class = 0x00
	Application           = 0x40
	ContextSpecific       = 0x80
	Private               = 0xC0
)

func (c Class) String() string {
	switch c {
	case Universal:
		return "Universal"
	case Application:
		return "Application"
	case ContextSpecific:
		return "Context-Specific"
	case Private:
		return "Private"
	default:
		return "unknown class"
	}
}

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

func (c Construction) String() string {
	switch c {
	case Primitive:
		return "Primitive"
	case Constructed:
		return "Constructed"
	default:
		return "unknown construction"
	}
}

type Tag struct {
	class     Class
	construct Construction
	value     int
}

func (t Tag) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tag - %s %s %d", t.class, t.construct, t.value)

	var buf bytes.Buffer
	_, err := encodeTag(&buf, t)
	if err != nil {
		return sb.String()
	}

	b, err := buf.ReadByte()
	if err != nil {
		return sb.String()
	}

	fmt.Fprintf(&sb, " (0x%02X)", b)

	return sb.String()
}

func (t Tag) Equals(o Tag) bool {
	return t.class == o.class && t.construct == o.construct && t.value == o.value
}

func defaultTag(v any) (Tag, error) {
	// try and get underlying value of choice
	if c, ok := v.(Choice); ok {
		t, ok := c.Tag()
		if !ok {
			logger.Print("no choice tag")
			return Tag{}, fmt.Errorf("tag has not been set for choice")
		}
		logger.Print("returning choice tag")
		return t, nil
	}

	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	var t Tag
	switch rv.Kind() {
	case reflect.Bool:
		t = Tag{class: Universal, construct: Primitive, value: BoolUniversalTagVal}
	case reflect.Int:
		t = Tag{class: Universal, construct: Primitive, value: IntUniversalTagVal}
	case reflect.String:
		t = Tag{class: Universal, construct: Primitive, value: OctetStrUniversalTag}
	case reflect.Slice:
		// only allow byte slices
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			t = Tag{class: Universal, construct: Primitive, value: OctetStrUniversalTag}
		}
	case reflect.Struct:
		t = Tag{class: Universal, construct: Constructed, value: SeqUniversalTagVal}
	case reflect.Map:
		t = Tag{class: Universal, construct: Constructed, value: SetUniversalTagVal}
	}

	var zero Tag
	if t == zero {
		return zero, fmt.Errorf("unknown default tag for kind %s", rv.Kind())
	}

	return t, nil
}

type Choice interface {
	// Given a Tag, returns a pointer to the corresponding value,
	// returns an error if Tag does not match
	Choose(Tag) (any, error)
	// Returns the tag of the chosen value if set, or false if no value has been set
	Tag() (Tag, bool)
}

func Chosen(c Choice) (any, error) {
	t, ok := c.Tag()
	if !ok {
		return nil, fmt.Errorf("cannot get chosen: no choice has been made")
	}

	return c.Choose(t)
}

type BerStructTag struct {
	tag   *int
	class *Class
}

func NewBerStructTag(s string) (BerStructTag, error) {
	kv := map[string]string{}
	for _, s1 := range strings.Split(s, ",") {
		if s1 == "" {
			continue
		}

		spl := strings.Split(s1, "=")
		if len(spl) != 2 {
			return BerStructTag{}, fmt.Errorf("invalid tag value %q", s1)
		}

		kTrim := strings.TrimSpace(spl[0])
		vTrim := strings.TrimSpace(spl[1])

		kv[kTrim] = vTrim
	}

	var bst BerStructTag

	if tagStr, ok := kv["tag"]; ok {
		tag, err := strconv.Atoi(tagStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid tag value %q: %w", tagStr, err)
		}

		bst.tag = new(int)
		*bst.tag = tag
	}

	if classStr, ok := kv["class"]; ok {
		class, err := classFromStructTag(classStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid class value %q: %w", classStr, err)
		}

		bst.class = new(Class)
		*bst.class = class
	}

	return bst, nil
}

func encodeBool(w io.Writer, b bool) (int, error) {
	var v byte
	if b {
		v = 0xFF
	}

	n, err := w.Write([]byte{v})
	return n, err
}

func decodeBool(r io.Reader, len int) (bool, error) {
	if len != 1 {
		return false, fmt.Errorf("incorrect byte len (%d) for a boolean, expected 1", len)
	}

	br := bufio.NewReader(r)
	byt, err := br.ReadByte()
	if err != nil {
		return false, err
	}

	switch byt {
	case 0x00:
		return false, nil
	case 0xFF:
		return true, nil
	default:
		return false, fmt.Errorf("unknown byte value 0x%X", byt)
	}
}

func reduce(rep64 []byte, neg bool) []byte {
	start := 0

	// if the first byte and the first bit of the second byte are
	// either all 1s or 0s then the first byte is redundant and can be
	// ignored
	for i := 0; i < len(rep64)-1; i++ {
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

func encodeInt(w io.Writer, i int) (int, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	n, err := w.Write(reduce(b, i < 0))
	return n, err
}

func decodeInt(r io.Reader, len int) (int, error) {
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
	for i := 0; i < start; i++ {
		byteRep[i] = twos
	}

	i := int(binary.BigEndian.Uint64(byteRep))
	return i, nil
}

func decodeOctetString(r io.Reader, len int) ([]byte, error) {
	b := make([]byte, len)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func encodeTag(w io.Writer, t Tag) (int, error) {
	id := byte(t.class) | byte(t.construct)
	if t.value >= 0x1F {
		return 0, fmt.Errorf("tag val is %d, multibyte identifiers unsupported", t.value)
	}

	id |= byte(t.value)

	return w.Write([]byte{id})
}

func decodeTag(r io.Reader) (Tag, error) {
	b1 := []byte{0}
	if _, err := io.ReadFull(r, b1); err != nil {
		return Tag{}, err
	}
	b := b1[0]

	class := Class(b & 0xC0)
	cons := Construction(b & 0x20)
	val := int(b & 0x1F)
	if val == 31 {
		return Tag{}, fmt.Errorf("Multibyte tag values unsupported")
	}

	return Tag{class, cons, val}, nil
}

func encodeLen(w io.Writer, len int) (int, error) {
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

func decodeLen(r io.Reader) (int, error) {
	b1 := []byte{0}
	if _, err := io.ReadFull(r, b1); err != nil {
		return 0, err
	}

	b := b1[0]
	if b < 128 {
		return int(b), nil
	}

	lenlen := int(b & 0x7F)
	if lenlen > 8 {
		return 0, fmt.Errorf("length is too big to represent in an int64: %d", lenlen)
	}
	buf := make([]byte, 8)

	if _, err := io.ReadFull(r, buf[8-lenlen:]); err != nil {
		return 0, err
	}

	ui := binary.BigEndian.Uint64(buf)
	return int(ui), nil
}

func tagWithBerStruct(v any, bst BerStructTag) (Tag, error) {
	t, err := defaultTag(v)
	if err != nil {
		return Tag{}, err
	}

	if bst.class != nil {
		t.class = *bst.class
	}

	if bst.tag != nil {
		t.value = *bst.tag
	}

	return t, nil
}

// seq is a pointer to an interface which is a struct containing only exported encodable values
func encodeSequence(w io.Writer, seq any) (int, error) {
	v := reflect.ValueOf(seq)
	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("seq is not a struct: %q", v.Kind())
	}

	written := 0
	for i := range v.NumField() {
		f := v.Field(i)
		logger.Printf("field %d (%s) is %s", i, v.Type().Field(i).Name, f.Type().Name())

		bst, err := NewBerStructTag(v.Type().Field(i).Tag.Get("ber"))

		if !f.CanInterface() {
			return 0, fmt.Errorf("cannot get interface value for %q (field is probably unexported)", v.Type().Field(i).Name)
		}

		logger.Printf("field %d value is %v", i, f.Interface())

		t, err := tagWithBerStruct(f.Interface(), bst)
		if err != nil {
			return written, err
		}

		n, err := encodeTlv(w, t, f.Interface())
		written += n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

// seq is a pointer to a bervalue struct
func decodeSequence(r io.Reader, len int, seq any) error {
	v := reflect.ValueOf(seq)
	logger.Print(v.Kind())
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("seq is not a pointer or is nil")
	}

	elm := v.Elem()
	logger.Print(elm.Kind())
	if elm.Kind() != reflect.Struct {
		return fmt.Errorf("seq %q does not point to a struct", elm.Kind())
	}

	// loop through all the fields of the struct, setting the zero
	// value and then calling Decode for the field
	for i := range elm.Type().NumField() {
		f := elm.Field(i)
		logger.Print(f.Kind())
		logger.Print(f.Type().Name())

		if f.Kind() == reflect.Pointer && f.CanInterface() {
			if _, ok := f.Interface().(Choice); ok {
				logger.Print("i is a choice")
				f.Set(reflect.New(f.Type().Elem()))

				c := f.Interface().(Choice)
				if err := decodeWithChoice(r, c); err != nil {
					return err
				}
				continue
			}
		}

		bst, err := NewBerStructTag(elm.Type().Field(i).Tag.Get("ber"))
		if err != nil {
			return err
		}

		if !f.CanAddr() {
			return fmt.Errorf("cannot addr %s (field probably unexported)", f.Type().Name())
		}

		i := f.Addr().Interface()

		t, err := tagWithBerStruct(i, bst)
		if err != nil {
			return err
		}

		if err := DecodeWithTag(r, t, i); err != nil {
			// TODO maybe wrap?
			return err
		}
	}

	return nil
}

func encodeChoice(w io.Writer, choice Choice) (int, error) {
	t, ok := choice.Tag()
	if !ok {
		return 0, fmt.Errorf("not tag set for choice value")
	}

	i, err := choice.Choose(t)
	if err != nil {
		return 0, err
	}

	return encodeTlv(w, t, i)
}

func encodeContents(w io.Writer, contents any) (int, error) {
	rc := reflect.ValueOf(contents)

	logger.Printf("contents kind is: %q", rc.Kind())

	// dereference all pointers
	if rc.Kind() == reflect.Pointer {
		rc = rc.Elem()
		logger.Printf("contents is a pointer, dereferencing to %q", rc.Kind())
	}

	if !rc.CanInterface() {
		return 0, fmt.Errorf("cannot get interface value for %s", rc.Kind())
	}

	v := rc.Interface()

	switch rc.Kind() {
	case reflect.Bool:
		logger.Print("encoding bool")
		return encodeBool(w, v.(bool))
	case reflect.Int:
		logger.Print("encoding int")
		return encodeInt(w, v.(int))
	case reflect.String:
		logger.Print("encoding string")
		return w.Write([]byte(v.(string)))
	case reflect.Slice:
		if b, ok := v.([]byte); ok {
			logger.Print("encoding slice")
			return w.Write(b)
		}
	case reflect.Struct:
		logger.Print("encoding seq")
		return encodeSequence(w, v)
	}

	return 0, fmt.Errorf("unknown encoding method for kind %s", rc.Kind())
}

func decodeContents(r io.Reader, len int, contents any) error {
	v := reflect.ValueOf(contents)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("contents is not a pointer or is nil %s", v.Kind())
	}

	switch v.Elem().Kind() {
	case reflect.Bool:
		b, err := decodeBool(r, len)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(b))

	case reflect.Int:
		i, err := decodeInt(r, len)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(i))

	case reflect.String:
		b, err := decodeOctetString(r, len)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(string(b)))

	case reflect.Slice:
		if v.Elem().Type().Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("cannot decode slice type for []%s", v.Elem().Type().Elem().Kind())
		}
		b, err := decodeOctetString(r, len)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(b))

	case reflect.Struct:
		if err := decodeSequence(r, len, contents); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown decoding kind %q", v.Elem().Kind())
	}

	return nil
}

func encodeTlv(w io.Writer, tag Tag, contents any) (int, error) {
	logger.Print(contents)

	written := 0
	n, err := encodeTag(w, tag)
	written += n
	if err != nil {
		return written, err
	}

	if c, ok := contents.(Choice); ok {
		logger.Print("dereferencing choice")
		v, err := Chosen(c)
		if err != nil {
			return written, err
		}
		contents = v
	}

	var buf bytes.Buffer
	contentsLen, err := encodeContents(&buf, contents)
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

func Encode(w io.Writer, v any) (int, error) {
	tag, err := defaultTag(v)
	if err != nil {
		return 0, err
	}

	return encodeTlv(w, tag, v)
}

func decodeWithChoice(r io.Reader, c Choice) error {
	dt, err := decodeTag(r)
	if err != nil {
		return err
	}

	v, err := c.Choose(dt)
	if err != nil {
		return err
	}

	len, err := decodeLen(r)
	if err != nil {
		return fmt.Errorf("error decoding len: %w", err)
	}

	return decodeContents(r, len, v)
}

func DecodeWithTag(r io.Reader, t Tag, v any) error {
	dt, err := decodeTag(r)
	if err != nil {
		return fmt.Errorf("error decoding tag: %w", err)
	}

	logger.Print(dt)

	if t.class != dt.class {
		return fmt.Errorf("tag class 0x%X does not match decoded 0x%X", t.class, dt.class)
	}

	if t.construct != dt.construct {
		return fmt.Errorf("tag construction 0x%X does not match decoded 0x%X", t.construct, dt.construct)
	}

	if t.value != dt.value {
		return fmt.Errorf("tag value %d does not match decoded %d", t.value, dt.value)
	}

	len, err := decodeLen(r)
	if err != nil {
		return fmt.Errorf("error decoding len: %w", err)
	}

	return decodeContents(r, len, v)
}

func Decode(r io.Reader, v any) error {
	def, err := defaultTag(v)
	if err != nil {
		return err
	}

	return DecodeWithTag(r, def, v)
}
