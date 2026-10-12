package official

import "testing"

func TestRegistry(t *testing.T) {
	names := map[string]bool{}
	datasets := map[string]bool{}
	for _, e := range Registry {
		if names[e.Name] {
			t.Errorf("duplicate name %q", e.Name)
		}
		names[e.Name] = true
		ds := e.New(nil).Dataset()
		if ds.ID == "" || ds.Name == "" || datasets[string(ds.ID)] {
			t.Errorf("%s: dataset %+v", e.Name, ds)
		}
		datasets[string(ds.ID)] = true
	}
}
