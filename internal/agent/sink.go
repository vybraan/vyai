package agent

import (
	"os"

	"github.com/grahms/promptweaver"
)

func BuildSink(uiOut func(string), workspace string) *promptweaver.HandlerSink {
	sink := promptweaver.NewHandlerSink()

	// Hidden reasoning
	sink.RegisterHandler("think", func(ev promptweaver.SectionEvent) {})

	// Shell execution
	sink.RegisterHandler("run-bash", func(ev promptweaver.SectionEvent) {
		if !AllowCommand(ev.Content) {
			uiOut("Blocked command: " + ev.Content)
			return
		}

		out, err := RunBash(ev.Content)
		if err != nil {
			uiOut("Exec error: " + err.Error())
			return
		}

		uiOut(out)
	})

	// File creation
	sink.RegisterHandler("create-file", func(ev promptweaver.SectionEvent) {
		path, err := SecureJoin(workspace, ev.Attrs["path"])
		if err != nil {
			uiOut("File blocked: " + err.Error())
			return
		}

		if err := os.WriteFile(path, []byte(ev.Content), 0644); err != nil {
			uiOut("Write failed: " + err.Error())
			return
		}

		uiOut("File created: " + path)
	})

	// output finale
	sink.RegisterHandler("summary", func(ev promptweaver.SectionEvent) {
		uiOut(ev.Content)
	})

	return sink
}
