package spy

import "testing"

func TestDiff(t *testing.T) {
	d := diff([]string{"foo"}, []string{"foobar"})
	if d[0] != "foo" {
		t.Errorf("diff(foo, foobar) = %s; want foo", d)
	}
}
