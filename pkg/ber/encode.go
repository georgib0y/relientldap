package ber

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func encodeBool(w io.Writer, b bool) (int, error) {
	var v byte
	if b {
		v = 0xFF
	}

	n, err := w.Write([]byte{v})
	return n, err
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

func encodeTag(w io.Writer, t Tag) (int, error) {
	id := byte(t.Class) | byte(t.Construct)
	if t.Value >= 0x1F {
		return 0, fmt.Errorf("tag val is %d, multibyte identifiers unsupported", t.Value)
	}

	id |= byte(t.Value)

	return w.Write([]byte{id})
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

func encodeStruct(w io.Writer, s any) (int, error) {
	if reflect.ValueOf(s).Kind() != reflect.Pointer {
		return 0, fmt.Errorf("encoding struct requies pointer not a %q", reflect.ValueOf(s).Kind())
	}

	if reflect.ValueOf(s).Elem().Kind() != reflect.Interface {
		return 0, fmt.Errorf("encoding pointer does not point to interface: %q", reflect.ValueOf(s).Elem().Kind())
	}

	v := reflect.ValueOf(s).Elem().Elem()
	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("encoding interface is not a struct: %q", v.Kind())
	}

	written := 0
	for i := range v.NumField() {
		f := v.Field(i)

		if !f.CanInterface() {
			return written, fmt.Errorf("cannot get value for %q, field may be unexported", v.Type().Field(i).Name)
		}

		st := v.Type().Field(i).Tag.Get("ber")
		tag, err := tagWithBerStruct(f.Interface(), st)
		if err != nil {
			return written, err
		}

		n, err := encodeTlv(w, tag, f.Interface())
		written += n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

func encodeContents(w io.Writer, contents any) (int, error) {
	rc := reflect.ValueOf(contents)

	logger.Printf("contents kind is: %q", rc.Kind())

	// dereference all pointers
	if rc.Kind() == reflect.Pointer {
		rc = rc.Elem()
		logger.Printf("contents is a pointer, dereferencing to %q", rc.Kind())
	}

	if rc.Kind() == reflect.Interface {
		rc = rc.Elem()
		logger.Printf("contents is an interface, dereferencing to %q", rc.Kind())
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
		logger.Print("encoding struct")
		return encodeStruct(w, &v)
	}

	return 0, fmt.Errorf("unknown encoding method for kind %s", rc.Kind())
}

func encodeTlv(w io.Writer, tag Tag, contents any) (int, error) {
	if c, ok := contents.(choice); ok {
		t, v, ok := c.chosen()
		if !ok {
			return 0, fmt.Errorf("choice is empty")
		}
		tag = t
		contents = v
	}

	if o, ok := contents.(optional); ok {
		v, ok := o.getAny()
		if !ok {
			return 0, nil
		}

		contents = v
	}

	written := 0
	n, err := encodeTag(w, tag)
	written += n
	if err != nil {
		return written, err
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
