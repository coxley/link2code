package main

import "testing"

func TestSplitFilename(t *testing.T) {
	testcases := []struct {
		text          string
		filename      string
		start         int
		end           int
		colonFilename bool
	}{
		{"path/to/file.txt", "path/to/file.txt", 0, 0, false},
		{"path/to/file.txt:1", "path/to/file.txt", 1, 0, false},
		{"path/to/file.txt:1-100", "path/to/file.txt", 1, 100, false},
		{"path/to/file.txt:1:20", "path/to/file.txt", 1, 0, false},
		{
			"foo/bar/baz/qux.go:3:import (",
			"foo/bar/baz/qux.go",
			3,
			0,
			false,
		},
		{"path/to/t:123/file.txt:1:20", "path/to/t:123/file.txt", 1, 0, true},
	}

	for _, tc := range testcases {
		filename, start, end := splitFilename(tc.text, tc.colonFilename)
		if tc.filename != filename {
			t.Fatalf("[%s] expected %s, got %s", tc.text, tc.filename, filename)
		}
		if tc.start != start {
			t.Fatalf("[%s] expected %d, got %d", tc.text, tc.start, start)
		}
		if tc.end != end {
			t.Fatalf("[%s] expected %d, got %d", tc.text, tc.end, end)
		}
	}
}
