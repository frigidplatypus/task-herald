package taskwarrior

import (
	"strings"

	"task-herald/internal/config"
)

// ModifyTask runs a Taskwarrior modify command for a given UUID and arguments.
// It checks syntax, logs errors, and verifies the modification succeeded.
// Returns true if the modification succeeded, false otherwise.
func ModifyTask(uuid string, args ...string) bool {
	if strings.TrimSpace(uuid) == "" {
		config.Log(config.ERROR, "ModifyTask: empty UUID")
		return false
	}
	cmdArgs := append([]string{uuid, "modify"}, args...)
	cmd := execCommand("task", cmdArgs...)
	cmdStr := "task " + strings.Join(cmdArgs, " ")
	config.Log(config.DEBUG, "ModifyTask running: %s", cmdStr)
	output, err := cmd.CombinedOutput()
	config.Log(config.DEBUG, "ModifyTask response: %s", string(output))
	if err != nil {
		config.Log(config.ERROR, "ModifyTask failed: %s\nError: %v\nOutput: %s", cmdStr, err, string(output))
		return false
	}
	// Check output for confirmation of modification
	if !strings.Contains(string(output), "Modified") && !strings.Contains(string(output), "modification") {
		config.Log(config.WARN, "ModifyTask: command ran but did not confirm modification: %s\nOutput: %s", cmdStr, string(output))
		return false
	}
	config.Log(config.INFO, "ModifyTask succeeded: %s", cmdStr)
	return true
}
