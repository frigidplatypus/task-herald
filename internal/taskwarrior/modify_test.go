package taskwarrior

import (
	"os/exec"
	"testing"
)

func TestModifyTask_EmptyUUID(t *testing.T) {
	ok := ModifyTask("   ")
	if ok {
		t.Error("expected false for empty UUID")
	}
}

func TestModifyTask_CommandFailure(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		// simulate a failing command
		return exec.Command("false")
	}

	ok := ModifyTask("uuid-1", "+tag")
	if ok {
		t.Error("expected ModifyTask to return false on command failure")
	}
}

func TestModifyTask_NoConfirmation(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	// command succeeds but output doesn't contain confirmation keywords
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "some output without any confirmation keywords")
	}

	ok := ModifyTask("uuid-2", "+tag")
	if ok {
		t.Error("expected ModifyTask to return false when output lacks confirmation")
	}
}

func TestModifyTask_Success(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	// command echoes a string that includes "Modified"
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "Modified 1 task")
	}

	ok := ModifyTask("uuid-3", "+tag")
	if !ok {
		t.Error("expected ModifyTask to return true on successful modification")
	}
}
