package hub

import "testing"

func TestUniqueName(t *testing.T) {
	// New clashes read "iPhone (2)"; a number used the older way ("iphone 2")
	// is not given again.
	devs := []string{"iPhone", "iphone 2", "Mac", "iPad", "ipad (2)"}
	for in, want := range map[string]string{"iPhone": "iPhone (3)", "Mac": "Mac (2)", "iPod": "iPod", "IPHONE": "IPHONE (3)", "iPad": "iPad (3)"} {
		if got := uniqueName(in, devs); got != want {
			t.Errorf("uniqueName(%q) = %q, want %q", in, got, want)
		}
	}
}
