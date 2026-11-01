package display

import "testing"

func TestDirOfCommand(t *testing.T) {
	for in, want := range map[string]string{
		`"C:\Program Files\Synergy\synergy-daemon.exe" --flag`: `C:\Program Files\Synergy`,
		`C:\Program Files\Synergy\synergy-daemon.exe --flag`:   `C:\Program Files\Synergy`,
		`C:\x\a.exe`: `C:\x`,
		``:           ``,
	} {
		if got := dirOfCommand(in); got != want {
			t.Errorf("dirOfCommand(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSynergyNoPanic(t *testing.T) {
	st, dir := Synergy()
	t.Logf("state=%d dir=%q major=%d", st, dir, SynergyMajor(dir))
}
