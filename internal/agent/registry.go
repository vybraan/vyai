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
	reg.Register(promptweaver.SectionPlugin{Name: "summary"})

	return reg
}
