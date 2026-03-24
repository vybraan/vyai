package agent

import "github.com/grahms/promptweaver"

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
