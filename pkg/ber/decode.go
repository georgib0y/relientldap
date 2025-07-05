package ber

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

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

// returns true if the value was decoded, or false if not decode (optional was not present)
func decodeStructField(r io.Reader, sv reflect.Value, idx int, dt Tag, len int) (bool, error) {
	f := sv.Field(idx)
	logger.Printf("%s: %s (%s)", sv.Type(), sv.Type().Field(idx).Name, f.Type())

	if !f.CanSet() || !f.CanInterface() {
		return false, fmt.Errorf("can't set or interface field %q (porbably unexported)", sv.Type().Field(idx).Name)
	}

	if _, ok := f.Interface().(choice); ok {
		f.Set(reflect.New(f.Type().Elem()))
		c := f.Interface().(choice)
		if err := decodeChoice(r, c, dt, len); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, ok := f.Interface().(optional); ok {
		f.Set(reflect.New(f.Type().Elem()))
		o := f.Interface().(optional)
		val, _ := o.getAny()
		t, err := tagWithBerStruct(val, sv.Type().Field(idx).Tag.Get("ber"))
		if err != nil {
			return false, err
		}

		if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
			logger.Print("finished decoding optional (no tag match)")
			return false, nil
		}
	}

	if !f.CanAddr() {
		return false, fmt.Errorf("cannot addr %s (field probably unexported)", f.Type().Name())
	}

	val := f.Addr().Interface()
	t, err := tagWithBerStruct(val, sv.Type().Field(idx).Tag.Get("ber"))
	if err != nil {
		return false, err
	}

	if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
		logger.Print("finished decoding optional (no tag match)")
		return false, fmt.Errorf("decoded tag %s did not match expected %s", dt, t)
	}

	return true, decodeContents(r, len, val)
}

func decodeStruct(r io.Reader, s any) error {
	v := reflect.ValueOf(s)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("s is not a pointer or is nil")
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("s does not point to a struct (%s)", v.Type())
	}

	if v.NumField() == 0 {
		return nil
	}

	dt, err := decodeTag(r)
	if err != nil {
		return err
	}
	len, err := decodeLen(r)
	if err != nil {
		return err
	}

	for i := range v.NumField() {
		decoded, err := decodeStructField(r, v, i, dt, len)
		if err != nil {
			return err
		}

		if decoded {
			dt1, err := decodeTag(r)
			if err != nil {
				return err
			}
			dt = dt1

			len1, err := decodeLen(r)
			if err != nil {
				return err
			}
			len = len1
		}
	}

	return nil
}

func decodeContents(r io.Reader, len int, contents any) error {
	v := reflect.ValueOf(contents)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("contents is not a pointer or is nil: %s", v.Kind())
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
		if err := decodeStruct(r, contents); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown decoding kind %q", v.Elem().Kind())
	}

	return nil
}

func decodeChoice(r io.Reader, c choice, dt Tag, len int) error {
	v, err := c.choose(dt)
	if err != nil {
		return err
	}

	return decodeContents(r, len, v)
}

func decodeOptional(r io.Reader, o optional, bstStr string) error {
	dt, err := decodeTag(r)
	if errors.Is(err, io.EOF) {
		// if no more bytes, this optional must be the last in the struct
		return nil
	} else if err != nil {
		return fmt.Errorf("error decoding tag: %w", err)
	}

	logger.Print(dt)

	if reflect.ValueOf(o).IsNil() {
		return fmt.Errorf("optional %s is nil", reflect.TypeOf(o))
	}

	val, _ := o.getAny()

	v := reflect.New(reflect.TypeOf(val))
	val = v.Interface()
	logger.Printf("optional type: %s", v.Type())

	def, err := defaultTag(val)
	if err != nil {
		return err
	}

	t, err := tagWithBerStruct(def, bstStr)
	if err != nil {
		return err
	}

	if t.Class != dt.Class || t.Construct != dt.Construct || t.Value != dt.Value {
		logger.Print("finished decoding optional (no tag match)")
		return nil
	}

	len, err := decodeLen(r)
	if err != nil {
		return fmt.Errorf("error decoding len: %w", err)
	}

	if err := decodeContents(r, len, val); err != nil {
		return err
	}

	o.setAny(val)

	logger.Printf("finished decoding optional %s ", reflect.TypeOf(val))
	return nil
}

func DecodeWithTag(r io.Reader, t Tag, v any) error {
	logger.Printf("decoding %s ", reflect.TypeOf(v))
	dt, err := decodeTag(r)
	if err != nil {
		return fmt.Errorf("error decoding tag: %w", err)
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
		return err
	}

	len, err := decodeLen(r)
	if err != nil {
		return fmt.Errorf("error decoding len: %w", err)
	}

	if err = decodeContents(r, len, v); err != nil {
		return err
	}

	logger.Printf("finished decoding %s ", reflect.TypeOf(v))
	return nil
}

func Decode(r io.Reader, v any) error {
	def, err := defaultTag(v)
	if err != nil {
		return err
	}

	return DecodeWithTag(r, def, v)
}
