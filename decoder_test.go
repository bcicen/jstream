package jstream

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
)

func mkReader(s string) *bytes.Reader { return bytes.NewReader([]byte(s)) }

func TestDecoderIntegerRounding(t *testing.T) {
	tests := []struct {
		name, number string
	}{
		{"zero", "0"},
		{"small", "12345"},
		{"belowExactLimit", "9007199254740991"},
		{"exactLimit", "9007199254740992"},
		{"aboveExactLimit", "9007199254740993"},
		{"oddAboveExactLimit", "9007199254740995"},
		{"reportedID", "144126219108390099"},
		{"signed64Limit", "9223372036854775807"},
		{"unsigned64Limit", "18446744073709551615"},
		{"wideInteger", "12345678901234567890"},
		{"longInteger", strings.Repeat("1234567890", 10)},
		{"largePowerOfTen", "1" + strings.Repeat("0", 100)},
	}
	for _, tt := range tests {
		for _, prefix := range []string{"", "-"} {
			number := prefix + tt.number
			t.Run(prefix+tt.name, func(t *testing.T) {
				var want float64
				if err := json.Unmarshal([]byte(number), &want); err != nil {
					t.Fatal(err)
				}
				decoder := NewDecoder(mkReader("["+number+"]"), 1)
				count := 0
				for mv := range decoder.Stream() {
					count++
					got, ok := mv.Value.(float64)
					if !ok || mv.ValueType != Number {
						t.Fatalf("got %v (%v), expected a number", mv.Value, mv.ValueType)
					}
					if math.Float64bits(got) != math.Float64bits(want) {
						t.Errorf("%s: got %.17g, expected %.17g", number, got, want)
					}
				}
				if err := decoder.Err(); err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Fatalf("got %d values, expected 1", count)
				}
			})
		}
	}
}

func TestDecoderIntegerOverflow(t *testing.T) {
	for _, prefix := range []string{"", "-"} {
		decoder := NewDecoder(mkReader("["+prefix+strings.Repeat("9", 400)+"]"), 1)
		for range decoder.Stream() {
		}
		err, ok := decoder.Err().(*strconv.NumError)
		if !ok || err.Err != strconv.ErrRange {
			t.Errorf("prefix %q: got %v, expected a numeric range error", prefix, decoder.Err())
		}
	}
}

func TestDecoderSimple(t *testing.T) {
	var (
		counter int
		mv      *MetaValue
		body    = `[{"bio":"bada bing bada boom","id":1,"name":"Charles","falseVal":false}]`
	)

	decoder := NewDecoder(mkReader(body), 1)

	for mv = range decoder.Stream() {
		counter++
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}

	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
}

func TestDecoderNested(t *testing.T) {
	var (
		counter int
		mv      *MetaValue
		body    = `{
  "1": {
    "bio": "bada bing bada boom",
    "id": 0,
    "name": "Roberto",
    "nested1": {
      "bio": "utf16 surrogate (\ud834\udcb2)\n\u201cutf 8\u201d",
      "id": 1.5,
      "name": "Roberto*Maestro",
      "nested2": { "nested2arr": [0,1,2], "nested3": {
        "nested4": { "depth": "recursion" }}
			}
		}
  },
  "2": {
    "nullfield": null,
    "id": -2
  }
}`
	)

	decoder := NewDecoder(mkReader(body), 2)

	for mv = range decoder.Stream() {
		counter++
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}

	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
}

func TestDecoderFlat(t *testing.T) {
	var (
		counter int
		mv      *MetaValue
		body    = `[
  "1st test string",
  "Roberto*Maestro", "Charles",
  0, null, false,
  1, 2.5
]`
		expected = []struct {
			Value     interface{}
			ValueType ValueType
		}{
			{
				"1st test string",
				String,
			},
			{
				"Roberto*Maestro",
				String,
			},
			{
				"Charles",
				String,
			},
			{
				0.0,
				Number,
			},
			{
				nil,
				Null,
			},
			{
				false,
				Boolean,
			},
			{
				1.0,
				Number,
			},
			{
				2.5,
				Number,
			},
		}
	)

	decoder := NewDecoder(mkReader(body), 1)

	for mv = range decoder.Stream() {
		if mv.Value != expected[counter].Value {
			t.Fatalf("got %v, expected: %v", mv.Value, expected[counter])
		}
		if mv.ValueType != expected[counter].ValueType {
			t.Fatalf("got %v value type, expected: %v value type", mv.ValueType, expected[counter].ValueType)
		}
		counter++
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}

	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
}

func TestDecoderMultiDoc(t *testing.T) {
	var (
		counter int
		mv      *MetaValue
		body    = `{ "bio": "bada bing bada boom", "id": 1, "name": "Charles" }
{ "bio": "bada bing bada boom", "id": 2, "name": "Charles" }
{ "bio": "bada bing bada boom", "id": 3, "name": "Charles" }
{ "bio": "bada bing bada boom", "id": 4, "name": "Charles" }
{ "bio": "bada bing bada boom", "id": 5, "name": "Charles" }
`
	)

	decoder := NewDecoder(mkReader(body), 0)

	for mv = range decoder.Stream() {
		if mv.ValueType != Object {
			t.Fatalf("got %v value type, expected: Object value type", mv.ValueType)
		}
		counter++
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}
	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
	if counter != 5 {
		t.Fatalf("expected 5 items, got %d", counter)
	}

	// test at depth level 1
	counter = 0
	kvcounter := 0
	decoder = NewDecoder(mkReader(body), 1)

	for mv = range decoder.Stream() {
		switch mv.Value.(type) {
		case KV:
			kvcounter++
		default:
			counter++
		}
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}
	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
	if kvcounter != 0 {
		t.Fatalf("expected 0 keyvalue items, got %d", kvcounter)
	}
	if counter != 15 {
		t.Fatalf("expected 15 items, got %d", counter)
	}

	// test at depth level 1 w/ emitKV
	counter = 0
	kvcounter = 0
	decoder = NewDecoder(mkReader(body), 1).EmitKV()

	for mv = range decoder.Stream() {
		switch mv.Value.(type) {
		case KV:
			kvcounter++
		default:
			counter++
		}
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}
	if err := decoder.Err(); err != nil {
		t.Fatalf("decoder error: %s", err)
	}
	if kvcounter != 15 {
		t.Fatalf("expected 15 keyvalue items, got %d", kvcounter)
	}
	if counter != 0 {
		t.Fatalf("expected 0 items, got %d", counter)
	}
}

func TestDecoderReaderFailure(t *testing.T) {
	var (
		failAfter = 900
		mockData  = byte('[')
	)

	r := newMockReader(failAfter, mockData)
	decoder := NewDecoder(r, -1)

	for mv := range decoder.Stream() {
		t.Logf("depth=%d offset=%d len=%d (%v)", mv.Depth, mv.Offset, mv.Length, mv.Value)
	}

	err := decoder.Err()
	t.Logf("got error: %s", err)
	if err == nil {
		t.Fatalf("missing expected decoder error")
	}

	derr, ok := err.(DecoderError)
	if !ok {
		t.Fatalf("expected error of type DecoderError, got %T", err)
	}

	if derr.ReaderErr() == nil {
		t.Fatalf("missing expected underlying reader error")
	}
}
