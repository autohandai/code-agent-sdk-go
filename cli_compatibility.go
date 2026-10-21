package autohand

import "fmt"

func validateCLIArguments(args []string) error {
	unsupported := map[string]bool{"--agents-md": true, "--no-agents-md": true, "--agents-md-create": true, "--agents-md-auto-update": true, "--agents-md-path": true, "--auto-save-interval": true, "--compression-threshold": true, "--summarization-threshold": true, "--skills": true, "--skill-sources": true, "--install-missing-skills": true, "--permission-mode": true, "--permission-allow-list": true, "--permission-deny-list": true}
	switches := map[string]bool{"--bare": true, "--offline": true, "--yes": true, "--y": true, "--dry-run": true, "--debug": true, "--auto-commit": true, "-c": true, "--unrestricted": true, "--restricted": true, "--plan": true, "--answer-only": true, "--setup-only": true, "--no-idle-logout": true, "--auto-skill": true, "--learn": true, "--learn-update": true, "--project": true, "--permissions": true, "--settings": true, "--traces-on": true, "--traces-off": true, "--login": true, "--logout": true, "--patch": true, "--acp": true, "--sdk-resume-last": true, "--tmux": true, "--auto-mode": true, "--no-worktree": true, "--interactive-on-complete": true, "--setup": true, "--about": true, "--feedback": true, "--cc": true, "--context-compact": true, "--no-cc": true, "--no-context-compact": true, "--ephemeral": true}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if unsupported[argument] {
			return fmt.Errorf("%s is unsupported by the current CLI; use a supported CLI option or runtime RPC instead", argument)
		}
		if !switches[argument] {
			index++
		}
	}
	return nil
}
