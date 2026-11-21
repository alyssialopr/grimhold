package world

import "testing"

const testWorld = `{"start":"a","regions":[
 {"id":"a","name":"A","danger":1,"foes":["goblin"],"exits":["b"]},
 {"id":"b","name":"B","danger":2,"foes":["orc"],"exits":["a","c"]},
 {"id":"c","name":"C","danger":3,"foes":["ogre"],"exits":["b"]}]}`

func TestCanTravel(t *testing.T) {
	w, err := Parse([]byte(testWorld))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tests := []struct {
		name     string
		from, to string
		ok       bool
	}{
		{"direct exit", "a", "b", true},
		{"other exit", "b", "c", true},
		{"no direct path", "a", "c", false},
		{"unknown region", "a", "nowhere", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.CanTravel(tt.from, tt.to)
			if (err == nil) != tt.ok {
				t.Errorf("CanTravel(%q, %q) = %v, want ok=%v", tt.from, tt.to, err, tt.ok)
			}
		})
	}
}

func TestParseRejectsBadExit(t *testing.T) {
	_, err := Parse([]byte(`{"start":"a","regions":[{"id":"a","foes":["x"],"exits":["ghost"]}]}`))
	if err == nil {
		t.Fatal("expected an error for an unknown exit")
	}
}
