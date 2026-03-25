package agent

import "github.com/grahms/promptweaver"

// BuildRegistry creates a promptweaver.Registry populated with the standard section plugins used by the agent.
// The registry includes: "think"; "run-bash" (aliases "exec", "shell"); "create-file" (alias "write-file"); "read-file" (alias "view-file"); "list-dir" (alias "ls"); "grep-file" (alias "search-file"); "glob-file" (alias "find-file"); "edit-file" (alias "update-file"); and "summary".
func BuildRegistry() *promptweaver.Registry {
	reg := promptweaver.NewRegistry()

	reg.Register(promptweaver.SectionPlugin{Name: "think"})
	reg.Register(promptweaver.SectionPlugin{
		Name:    "run-bash",
		Aliases: []string{"exec", "shell"},
	})
	reg.Register(promptweaver.SectionPlugin{
		Name:    "create-file",
		Aliases: []string{"write-file"},
	})
	reg.Register(promptweaver.SectionPlugin{Name: "read-file", Aliases: []string{"view-file"}})
	reg.Register(promptweaver.SectionPlugin{Name: "list-dir", Aliases: []string{"ls"}})
	reg.Register(promptweaver.SectionPlugin{Name: "grep-file", Aliases: []string{"search-file"}})
	reg.Register(promptweaver.SectionPlugin{Name: "glob-file", Aliases: []string{"find-file"}})
	reg.Register(promptweaver.SectionPlugin{Name: "edit-file", Aliases: []string{"update-file"}})
	reg.Register(promptweaver.SectionPlugin{Name: "summary"})

	return reg
}
