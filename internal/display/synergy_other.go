//go:build !windows

package display

// Synergy 只有 Windows 版會偵測。
func Synergy() (SynergyState, string) { return SynergyNone, "" }

// SynergyMajor 只有 Windows 版會偵測。
func SynergyMajor(string) int { return 0 }
