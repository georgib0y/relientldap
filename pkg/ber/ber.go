package ber

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var logger = log.New(os.Stderr, "ber: ", log.Lshortfile)

func init() {
	_ = io.Discard
	logger.SetOutput(io.Discard)
}

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

func constructionFromString(s string) (Construction, error) {
	switch s {
	case "primitive":
		return Primitive, nil
	case "constructed":
		return Constructed, nil
	default:
		return Private, fmt.Errorf("unknown construction, struct tag is %s", s)
	}
}

type Tag struct {
	Class     Class
	Construct Construction
	Value     int
}

func (t Tag) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tag - %s %s %d", t.Class, t.Construct, t.Value)

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
	return t.Class == o.Class && t.Construct == o.Construct && t.Value == o.Value
}

func defaultTag(v any) (Tag, error) {
	// try and get underlying value of choice
	if c, ok := v.(choice); ok {
		t, _, ok := c.Chosen()
		if !ok {
			logger.Print("no choice tag")
			return Tag{}, fmt.Errorf("tag has not been set for choice")
		}
		logger.Print("returning choice tag")
		return t, nil
	}

	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Pointer && rv.CanInterface() {
		logger.Printf("default tag rv %q can interface", rv.Type())
		if o, ok := rv.Elem().Interface().(optional); ok {
			logger.Print("v is an optional")
			// not concerned with whether not not o is some, just need the
			// type information
			val, _ := o.getAny()
			// calling elem to get what the pointer points to
			rv = reflect.ValueOf(val)
		} else {
			rv = rv.Elem()
		}

	}

	var t Tag
	switch rv.Kind() {
	case reflect.Bool:
		t = Tag{Class: Universal, Construct: Primitive, Value: BoolUniversalTagVal}
	case reflect.Int:
		t = Tag{Class: Universal, Construct: Primitive, Value: IntUniversalTagVal}
	case reflect.String:
		t = Tag{Class: Universal, Construct: Primitive, Value: OctetStrUniversalTag}
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			t = Tag{Class: Universal, Construct: Primitive, Value: OctetStrUniversalTag}
		} else {
			t = Tag{Class: Universal, Construct: Constructed, Value: SeqUniversalTagVal}
		}
	case reflect.Struct:
		t = Tag{Class: Universal, Construct: Constructed, Value: SeqUniversalTagVal}
	case reflect.Map:
		t = Tag{Class: Universal, Construct: Constructed, Value: SetUniversalTagVal}
	}

	var zero Tag
	if t == zero {
		return zero, fmt.Errorf("unknown default tag for kind %s", rv.Kind())
	}

	return t, nil
}

type choice interface {
	Choose(t Tag) (any, error)
	Chosen() (Tag, any, bool)
}

// TODO maybe make a type alias for a pointer to choice?
type Choice[T any] struct {
	Choices T
	Tag     Tag
	Val     any
}

func NewChoice[T any]() *Choice[T] {
	return &Choice[T]{}
}

// Creates a new choice with value v, panics if tag and value are not supported by the choice type T
func NewChosen[T any, V any](t Tag, v V) *Choice[T] {
	var c Choice[T]

	chosen, err := c.Choose(t)
	if err != nil {
		logger.Panic(err)
	}

	// need to call Elem() twice here, becuase choose returns a pointer to c.Val, which is an interface{}
	// valueof(chosen).type == *interface{}
	// so valueof(chosen).elem().type == interface{}
	// so valueof(chosen).elem().elem().type == some type
	// then need to call elem() on that so that it gets the underlying type of the interface{}

	cptr, ok := chosen.(*V)
	if !ok {
		logger.Panicf("chosen %q is not a pointer to value's type %q", reflect.TypeOf(chosen), reflect.TypeOf(v))
	}

	*cptr = v

	logger.Printf("new chosen's value is: %s", c.Val)
	return &c
}

// returns a pointer to the value chosen by tag
func (c *Choice[T]) Choose(t Tag) (any, error) {
	v := reflect.ValueOf(&c.Choices).Elem()

	// TODO interface nullability??
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("choices is not a struct")
	}

	for i := range v.NumField() {
		// TODO get choice from struct tag
		bst, err := NewBerStructTag(v.Type().Field(i).Tag.Get("ber"))
		if err != nil {
			return nil, fmt.Errorf("unable to parse struct tag for field %q: %w", v.Type().Field(i).Name, err)
		}

		if bst.val == nil {
			return nil, fmt.Errorf("no choice value for field %q", v.Type().Field(i).Name)
		}

		// TODO if bst fully matches t
		if t.Value != *bst.val {
			continue
		}

		f := v.Field(i)
		if !f.CanAddr() {
			return nil, fmt.Errorf("field %q can not be addred", v.Type().Field(i).Name)
		}

		if f.Addr().CanSet() {
			return nil, fmt.Errorf("addr cannot be set for field %q", v.Type().Field(i).Name)
		}

		c.Tag = t
		c.Val = f.Addr().Interface()
		logger.Printf("chosen type is %s", reflect.TypeOf(c.Val))
		return c.Val, nil
	}

	return nil, fmt.Errorf("unknown tag %s for choices: %s", t, v.Type().Name())
}

// will return the tag and a pointer to the chosen value, or false if nothing has been set
func (c *Choice[T]) Chosen() (Tag, any, bool) {
	var zero Tag
	return c.Tag, c.Val, c.Tag != zero
}

type optional interface {
	// Returns a pointer to the option's value as an interface{}
	// the returned value should be a pointer to the zero value if unset, rather than nil so that
	// other functions may know the type of optional
	// TODO maybe make a zero() function that handles this?
	getAny() (any, bool)
	setAny(v any) error
}

type Optional[T any] struct {
	is_some bool
	val     T
}

func NewOptional[T any](v T) *Optional[T] {
	return &Optional[T]{is_some: true, val: v}
}

func NewEmpty[T any]() *Optional[T] {
	var zero T
	return &Optional[T]{is_some: false, val: zero}
}

func (o *Optional[T]) Get() (T, bool) {
	if o == nil {
		// TODO defaultTag() relies on getAny() to return some value that can be reflected on
		// ugly?
		var zero T
		return zero, false
	}
	return o.val, o.is_some
}

func (o *Optional[T]) getAny() (any, bool) {
	return o.Get()
}

func (o *Optional[T]) Set(v T) {
	o.val = v
	o.is_some = true
}

func (o *Optional[T]) setAny(v any) error {
	if reflect.TypeOf(v) != reflect.TypeFor[T]() {
		return fmt.Errorf("v's type %s does not match optional's %s", reflect.TypeOf(v), reflect.TypeFor[T]())
	}
	vt, ok := v.(T)
	if !ok {
		return fmt.Errorf("could not cast to %s", reflect.TypeFor[T]())
	}

	o.Set(vt)
	return nil
}

func (o *Optional[T]) Unset(v T) {
	o.is_some = false
}

type Set[T comparable] map[T]struct{}

type BerStructTag struct {
	class *Class
	cons  *Construction
	val   *int
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

	if classStr, ok := kv["class"]; ok {
		class, err := classFromStructTag(classStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid class value %q: %w", classStr, err)
		}

		bst.class = new(Class)
		*bst.class = class
	}

	if consStr, ok := kv["cons"]; ok {
		cons, err := constructionFromString(consStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid construction value %q: %w", consStr, err)
		}

		bst.cons = new(Construction)
		*bst.cons = cons
	}

	if valStr, ok := kv["val"]; ok {
		val, err := strconv.Atoi(valStr)
		if err != nil {
			return BerStructTag{}, fmt.Errorf("invalid tag value %q: %w", valStr, err)
		}

		bst.val = new(int)
		*bst.val = val
	}

	return bst, nil
}

func tagWithBerStruct(v any, st string) (Tag, error) {
	t, err := defaultTag(v)
	if err != nil {
		return t, err
	}

	bst, err := NewBerStructTag(st)
	if err != nil {
		return t, err
	}

	if bst.class != nil {
		t.Class = *bst.class
	}

	if bst.cons != nil {
		t.Construct = *bst.cons
	}

	if bst.val != nil {
		t.Value = *bst.val
	}

	return t, nil
}
