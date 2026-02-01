package template

import (
	"testing"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected int
		ok       bool
	}{
		{42, 42, true},
		{int64(100), 100, true},
		{float64(3.7), 3, true},
		{float32(5.5), 5, true},
		{"123", 123, true},
		{"456px", 456, true},
		{"abc", 0, false},
		{"", 0, false},
		{nil, 0, false},
		{true, 0, false},
		{[]int{1, 2}, 0, false},
	}

	for _, tt := range tests {
		got, ok := toInt(tt.input)
		if ok != tt.ok {
			t.Errorf("toInt(%v) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if got != tt.expected {
			t.Errorf("toInt(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{"<a href=\"test\">", "&lt;a href=&quot;test&quot;&gt;"},
		{"", ""},
		{"no special chars", "no special chars"},
	}

	for _, tt := range tests {
		got := escapeHTML(tt.input)
		if got != tt.expected {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildImgTag(t *testing.T) {
	tests := []struct {
		src      string
		args     ImageFilterArgs
		expected string
	}{
		{
			"image.png",
			ImageFilterArgs{},
			`<img src="image.png" />`,
		},
		{
			"image.png",
			ImageFilterArgs{Alt: "My Image"},
			`<img src="image.png" alt="My Image" />`,
		},
		{
			"image.png",
			ImageFilterArgs{Width: 100, Height: 200},
			`<img src="image.png" width="100" height="200" />`,
		},
		{
			"image.png",
			ImageFilterArgs{Class: "thumbnail"},
			`<img src="image.png" class="thumbnail" />`,
		},
		{
			"image.png",
			ImageFilterArgs{Alt: "Test", Width: 50, Height: 50, Class: "icon"},
			`<img src="image.png" alt="Test" width="50" height="50" class="icon" />`,
		},
		{
			"image.png",
			ImageFilterArgs{Alt: "<script>alert('xss')</script>"},
			`<img src="image.png" alt="&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" />`,
		},
	}

	for _, tt := range tests {
		got := buildImgTag(tt.src, tt.args)
		if got != tt.expected {
			t.Errorf("buildImgTag(%q, %+v) = %q, want %q", tt.src, tt.args, got, tt.expected)
		}
	}
}

func TestParseImageArgs_Positional(t *testing.T) {
	args := []interface{}{"data", "alt text", 100, 200, "my-class"}
	result := parseImageArgs(args)

	if result.Data != "data" {
		t.Errorf("Data = %v, want %v", result.Data, "data")
	}
	if result.Alt != "alt text" {
		t.Errorf("Alt = %v, want %v", result.Alt, "alt text")
	}
	if result.Width != 100 {
		t.Errorf("Width = %v, want %v", result.Width, 100)
	}
	if result.Height != 200 {
		t.Errorf("Height = %v, want %v", result.Height, 200)
	}
	if result.Class != "my-class" {
		t.Errorf("Class = %v, want %v", result.Class, "my-class")
	}
}

func TestParseImageArgs_Partial(t *testing.T) {
	args := []interface{}{"data", "alt text"}
	result := parseImageArgs(args)

	if result.Data != "data" {
		t.Errorf("Data = %v, want %v", result.Data, "data")
	}
	if result.Alt != "alt text" {
		t.Errorf("Alt = %v, want %v", result.Alt, "alt text")
	}
	if result.Width != 0 {
		t.Errorf("Width = %v, want 0", result.Width)
	}
	if result.Height != 0 {
		t.Errorf("Height = %v, want 0", result.Height)
	}
}

func TestParseImageArgs_Named(t *testing.T) {
	args := []interface{}{
		map[string]interface{}{
			"data":   "my data",
			"alt":    "my alt",
			"width":  150,
			"height": 250,
			"class":  "named-class",
		},
	}
	result := parseImageArgs(args)

	if result.Data != "my data" {
		t.Errorf("Data = %v, want %v", result.Data, "my data")
	}
	if result.Alt != "my alt" {
		t.Errorf("Alt = %v, want %v", result.Alt, "my alt")
	}
	if result.Width != 150 {
		t.Errorf("Width = %v, want %v", result.Width, 150)
	}
	if result.Height != 250 {
		t.Errorf("Height = %v, want %v", result.Height, 250)
	}
	if result.Class != "named-class" {
		t.Errorf("Class = %v, want %v", result.Class, "named-class")
	}
}

func TestParseImageArgs_StringDimensions(t *testing.T) {
	args := []interface{}{"data", "alt", "100", "200"}
	result := parseImageArgs(args)

	if result.Width != 100 {
		t.Errorf("Width = %v, want 100 (from string)", result.Width)
	}
	if result.Height != 200 {
		t.Errorf("Height = %v, want 200 (from string)", result.Height)
	}
}

func TestParseImageArgs_Empty(t *testing.T) {
	result := parseImageArgs(nil)

	if result.Data != nil {
		t.Errorf("Data = %v, want nil", result.Data)
	}
	if result.Alt != "" {
		t.Errorf("Alt = %v, want empty", result.Alt)
	}
	if result.Width != 0 {
		t.Errorf("Width = %v, want 0", result.Width)
	}
}

func TestJsonFilter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", `"hello"`},
		{123, "123"},
		{true, "true"},
		{nil, "null"},
		{[]int{1, 2, 3}, "[1,2,3]"},
		{map[string]int{"a": 1}, `{"a":1}`},
	}

	for _, tt := range tests {
		got, err := jsonFilter(tt.input)
		if err != nil {
			t.Errorf("jsonFilter(%v) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("jsonFilter(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestJsonFilter_Unmarshallable(t *testing.T) {
	// Channel can't be marshalled to JSON
	ch := make(chan int)
	got, err := jsonFilter(ch)
	if err != nil {
		t.Errorf("jsonFilter should not return error, got: %v", err)
	}
	if got != "null" {
		t.Errorf("jsonFilter for unmarshallable should return 'null', got: %v", got)
	}
}

func TestGenerateUID(t *testing.T) {
	uid1 := generateUID()
	uid2 := generateUID()
	uid3 := generateUID()

	if uid2 != uid1+1 {
		t.Errorf("UIDs should be sequential: %d, %d", uid1, uid2)
	}
	if uid3 != uid2+1 {
		t.Errorf("UIDs should be sequential: %d, %d", uid2, uid3)
	}
}
