// +build windows

package sshtool

// windows not support Terminal Resize
func (t *SSHTerminal) updateTerminalSize() {
	return
}
