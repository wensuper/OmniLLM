package hub

import "testing"

func TestUniqueName(t *testing.T) {
	devs := []string{"iPhone", "iphone 2", "Mac"}
	for in, want := range map[string]string{"iPhone": "iPhone 3", "Mac": "Mac 2", "iPad": "iPad", "IPHONE": "IPHONE 3"} {
		if got := uniqueName(in, devs); got != want {
			t.Errorf("uniqueName(%q) = %q, want %q", in, got, want)
		}
	}
}
