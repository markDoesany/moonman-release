package main

import "testing"

func TestResolveCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		open bool
		err  bool
	}{
		{name: "no arguments opens gui", args: nil, open: true},
		{name: "open opens gui", args: []string{"open"}, open: true},
		{name: "help exits without gui", args: []string{"--help"}},
		{name: "unknown command fails", args: []string{"package"}, err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			open, err := resolveCommand(test.args)
			if open != test.open {
				t.Fatalf("open = %t, want %t", open, test.open)
			}
			if (err != nil) != test.err {
				t.Fatalf("error = %v, want error = %t", err, test.err)
			}
		})
	}
}
