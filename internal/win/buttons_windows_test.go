package win

import "testing"

// The warm-up opens under the cursor that has just pressed a button of the
// reminder. If its own button were in the same place, a double click on
// "start" would put the warm-up straight off again.
func TestWarmupButtonIsClearOfTheReminder(t *testing.T) {
	w := &Window{dpi: 96, labels: Labels{Short: "Snooze 5 min"}}
	w.shown = view{kind: countdown}
	later := w.buttons()
	if len(later) != 1 || later[0].id != Short {
		t.Fatalf("warm-up buttons %+v, want the short snooze alone", later)
	}
	l := later[0].r
	if c := w.hit(int(l.left+l.right)/2, int(l.top+l.bottom)/2); c != Short {
		t.Fatalf("a click in the middle of the button gives %v", c)
	}
	for _, v := range []view{{kind: asking}, {kind: asking, ask: Ask{Long: true}}} {
		w.shown = v
		for _, b := range w.buttons() {
			if l.left < b.r.right && b.r.left < l.right && l.top < b.r.bottom && b.r.top < l.bottom {
				t.Fatalf("the warm-up button %+v overlaps the reminder's %+v", l, b.r)
			}
		}
	}
	w.shown = view{}
	if b := w.buttons(); b != nil {
		t.Fatalf("a hidden window has buttons: %+v", b)
	}
}
