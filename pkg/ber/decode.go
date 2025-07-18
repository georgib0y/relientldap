package ber

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

func decodeBool(r io.Reader, len int) (bool, int, error) {
	if len != 1 {
		return false, 0, fmt.Errorf("incorrect byte len (%d) for a boolean, expected 1", len)
	}

	buf := []byte{0}
	logger.Print("reading bool byte...")
	if _, err := io.ReadFull(r, buf); err != nil {
		return false, 0, err
	}
	byt := buf[0]
	logger.Print("read 1 bool byte")

	switch byt {
	case 0x00:
		return false, 1, nil
	case 0xFF:
		return true, 1, nil
	default:
		return false, 1, fmt.Errorf("unknown byte value 0x%X", byt)
	}
}

func decodeInt(r io.Reader, len int) (int, int, error) {
	if len > 8 {
		return 0, 0, fmt.Errorf("int len (%d) is too long", len)
	}

	byteRep := make([]byte, 8)
	start := 8 - len
	logger.Print("reading int bytes...")
	n, err := r.Read(byteRep[start:])
	if err != nil {
		return 0, n, err
	}
	logger.Printf("read %d int bytes", n)

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
	return i, n, nil
}

func decodeOctetString(r io.Reader, len int) ([]byte, int, error) {
	b := make([]byte, len)
	logger.Print("reading octet string bytes...")
	n, err := io.ReadFull(r, b)
	logger.Printf("read %d octet string bytes", n)
	if err != nil {
		return nil, n, err
	}

	return b, n, nil
}

func decodeTag(r io.Reader) (Tag, int, error) {
	b1 := []byte{0}
	logger.Print("reading tag bytes...")
	n, err := io.ReadFull(r, b1)
	if err != nil {
		return Tag{}, n, err
	}
	b := b1[0]
	logger.Printf("read %d tag bytes", n)

	class := Class(b & 0xC0)
	cons := Construction(b & 0x20)
	val := int(b & 0x1F)
	if val == 31 {
		return Tag{}, n, fmt.Errorf("Multibyte tag values unsupported")
	}

	return Tag{class, cons, val}, n, nil
}

func decodeLen(r io.Reader) (int, int, error) {
	b1 := []byte{0}
	logger.Print("reading len bytes...")
	n, err := io.ReadFull(r, b1)
	if err != nil {
		return 0, n, err
	}
	logger.Print("read 1 len bytes")

	b := b1[0]
	if b < 128 {
		return int(b), n, nil
	}

	lenlen := int(b & 0x7F)
	if lenlen > 8 {
		return 0, n, fmt.Errorf("length is too big to represent in an int64: %d", lenlen)
	}
	buf := make([]byte, 8)

	logger.Print("reading extend len bytes...")
	n1, err := io.ReadFull(r, buf[8-lenlen:])
	if err != nil {
		return 0, n + n1, err
	}
	logger.Printf("read %d extend len bytes", n1)

	ui := binary.BigEndian.Uint64(buf)
	return int(ui), n + n1, nil
}

// returns true if the value was decoded, or false if not decode (optional was not present)
func decodeStructField(r io.Reader, sv reflect.Value, idx int, dt Tag, len int) (bool, int, error) {
	f := sv.Field(idx)
	logger.Printf("%s: %s (%s)", sv.Type(), sv.Type().Field(idx).Name, f.Type())

	if !f.CanSet() || !f.CanInterface() {
		return false, 0, fmt.Errorf("can't set or interface field %q (porbably unexported)", sv.Type().Field(idx).Name)
	}

	if _, ok := f.Interface().(choice); ok {
		f.Set(reflect.New(f.Type().Elem()))
		c := f.Interface().(choice)
		n, err := decodeChoice(r, c, dt, len)
		if err != nil {
			return false, n, err
		}
		return true, n, nil
	}

	if _, ok := f.Interface().(optional); ok {
		f.Set(reflect.New(f.Type().Elem()))
		o := f.Interface().(optional)
		val, _ := o.getAny()
		t, err := tagWithBerStruct(val, sv.Type().Field(idx).Tag.Get("ber"))
		if err != nil {
			return false, 0, err
		}

		if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
			logger.Print("finished decoding optional (no tag match)")
			return false, 0, nil
		}
	}

	if !f.CanAddr() {
		return false, 0, fmt.Errorf("cannot addr %s (field probably unexported)", f.Type().Name())
	}

	val := f.Addr().Interface()
	t, err := tagWithBerStruct(val, sv.Type().Field(idx).Tag.Get("ber"))
	if err != nil {
		return false, 0, err
	}

	if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
		logger.Print("finished decoding optional (no tag match)")
		return false, 0, fmt.Errorf("decoded tag %s did not match expected %s", dt, t)
	}

	n, err := decodeContents(r, len, val)
	return true, n, err
}

func decodeStruct(r io.Reader, len int, s any) (int, error) {
	v := reflect.ValueOf(s)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return 0, fmt.Errorf("s is not a pointer or is nil")
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("s does not point to a struct (%s)", v.Type())
	}

	if v.NumField() == 0 {
		return 0, nil
	}

	read := 0

	dt, n, err := decodeTag(r)
	read += n
	if err != nil {
		return read, err
	}
	dl, n, err := decodeLen(r)
	read += n
	if err != nil {
		return read, err
	}

	logger.Printf("there are %d fields in struct %s", v.NumField(), v.Type())
	for i := range v.NumField() {

		decoded, n, err := decodeStructField(r, v, i, dt, dl)
		read += n
		if err != nil {
			return read, err
		}

		if read == len {
			break
		}

		if decoded {
			dt1, n, err := decodeTag(r)
			read += n
			if err != nil {
				return read, err
			}
			dt = dt1

			dl1, n, err := decodeLen(r)
			read += n
			if err != nil {
				return read, err
			}
			dl = dl1
		}
	}

	return read, nil
}

func decodeContents(r io.Reader, len int, contents any) (int, error) {
	v := reflect.ValueOf(contents)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return 0, fmt.Errorf("contents is not a pointer or is nil: %s", v.Kind())
	}

	read := 0

	switch v.Elem().Kind() {
	case reflect.Bool:
		b, n, err := decodeBool(r, len)
		read += n
		if err != nil {
			return read, err
		}
		v.Elem().Set(reflect.ValueOf(b))

	case reflect.Int:
		i, n, err := decodeInt(r, len)
		read += n
		if err != nil {
			return read, err
		}
		v.Elem().Set(reflect.ValueOf(i))

	case reflect.String:
		b, n, err := decodeOctetString(r, len)
		read += n
		if err != nil {
			return read, err
		}
		v.Elem().Set(reflect.ValueOf(string(b)))

	case reflect.Slice:
		if v.Elem().Type().Elem().Kind() != reflect.Uint8 {
			return 0, fmt.Errorf("cannot decode slice type for []%s", v.Elem().Type().Elem().Kind())
		}
		b, n, err := decodeOctetString(r, len)
		read += n
		if err != nil {
			return read, err
		}
		v.Elem().Set(reflect.ValueOf(b))
	case reflect.Struct:
		n, err := decodeStruct(r, len, contents)
		read += n
		if err != nil {
			return read, err
		}
	default:
		return 0, fmt.Errorf("unknown decoding kind %q", v.Elem().Kind())
	}

	return read, nil
}

func decodeChoice(r io.Reader, c choice, dt Tag, len int) (int, error) {
	logger.Printf("chosen choice tag is: %s", dt)
	v, err := c.Choose(dt)
	if err != nil {
		return 0, err
	}

	return decodeContents(r, len, v)
}

func decodeOptional(r io.Reader, o optional, bstStr string) (int, error) {
	read := 0

	dt, n, err := decodeTag(r)
	read += n
	if errors.Is(err, io.EOF) {
		// if no more bytes, this optional must be the last in the struct
		return read, nil
	} else if err != nil {
		return read, fmt.Errorf("error decoding tag: %w", err)
	}

	logger.Print(dt)

	if reflect.ValueOf(o).IsNil() {
		return read, fmt.Errorf("optional %s is nil", reflect.TypeOf(o))
	}

	val, _ := o.getAny()

	v := reflect.New(reflect.TypeOf(val))
	val = v.Interface()
	logger.Printf("optional type: %s", v.Type())

	def, err := defaultTag(val)
	if err != nil {
		return read, err
	}

	t, err := tagWithBerStruct(def, bstStr)
	if err != nil {
		return read, err
	}

	if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
		logger.Print("finished decoding optional (no tag match)")
		return read, nil
	}

	len, n, err := decodeLen(r)
	read += n
	if err != nil {
		return read, fmt.Errorf("error decoding len: %w", err)
	}

	n, err = decodeContents(r, len, val)
	read += n
	if err != nil {
		return read, err
	}

	o.setAny(val)

	logger.Printf("finished decoding optional %s ", reflect.TypeOf(val))
	return read, nil
}

func DecodeWithTag(r io.Reader, t Tag, v any) (int, error) {
	read := 0
	logger.Printf("decoding %s ", reflect.TypeOf(v))
	dt, n, err := decodeTag(r)
	read += n
	if err != nil {
		return read, fmt.Errorf("error decoding tag: %w", err)
	}

	logger.Print(dt)

	if t.Class != dt.Class {
		err = fmt.Errorf("tag class 0x%X does not match decoded 0x%X", t.Class, dt.Class)
	}

	if t.Construct != dt.Construct {
		err = fmt.Errorf("tag construction 0x%X does not match decoded 0x%X", t.Construct, dt.Construct)
	}

	if t.Value != dt.Value {
		err = fmt.Errorf("tag value %d does not match decoded %d", t.Value, dt.Value)
	}

	if err != nil {
		return read, err
	}

	len, n, err := decodeLen(r)
	read += n
	if err != nil {
		return read, fmt.Errorf("error decoding len: %w", err)
	}

	n, err = decodeContents(r, len, v)
	read += n
	if err != nil {
		return read, err
	}

	logger.Printf("finished decoding %s ", reflect.TypeOf(v))
	return read, nil
}

func Decode(r io.Reader, v any) error {
	logger.Print("in decode")
	def, err := defaultTag(v)
	if err != nil {
		return err
	}
	logger.Print("decoded default tag")

	_, err = DecodeWithTag(r, def, v)
	return err
}
