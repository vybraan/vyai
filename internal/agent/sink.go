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

	// File viewing
	sink.RegisterHandler("read-file", func(ev promptweaver.SectionEvent) {
		path, err := SecureJoin(workspace, ev.Attrs["path"])
		if err != nil {
			uiOut("File blocked: " + err.Error())
			return
		}
		out, err := ReadFile(path)
		if err != nil {
			uiOut("Read error: " + err.Error())
			return
		}
		uiOut(out)
	})

	// Directory listing
	sink.RegisterHandler("list-dir", func(ev promptweaver.SectionEvent) {
		path, err := SecureJoin(workspace, ev.Attrs["path"])
		if err != nil {
			uiOut("Path blocked: " + err.Error())
			return
		}
		out, err := ListDir(path)
		if err != nil {
			uiOut("List error: " + err.Error())
			return
		}
		uiOut(out)
	})

	// Grep file content
	sink.RegisterHandler("grep-file", func(ev promptweaver.SectionEvent) {
		out, err := GrepFile(ev.Attrs["pattern"], ev.Attrs["include"], ev.Attrs["path"])
		if err != nil {
			uiOut("Grep error: " + err.Error())
			return
		}
		uiOut(out)
	})

	// Glob file paths
	sink.RegisterHandler("glob-file", func(ev promptweaver.SectionEvent) {
		out, err := GlobFile(ev.Attrs["pattern"], ev.Attrs["path"])
		if err != nil {
			uiOut("Glob error: " + err.Error())
			return
		}
		uiOut(out)
	})

	// File editing
	sink.RegisterHandler("edit-file", func(ev promptweaver.SectionEvent) {
		path, err := SecureJoin(workspace, ev.Attrs["path"])
		if err != nil {
			uiOut("File blocked: " + err.Error())
			return
		}
		oldString := ev.Attrs["old"]
		newString := ev.Attrs["new"]

		if err := EditFile(path, oldString, newString); err != nil {
			uiOut("Edit failed: " + err.Error())
			return
		}
		uiOut("File edited: " + path)
	})

	// output finale
	sink.RegisterHandler("summary", func(ev promptweaver.SectionEvent) {
		uiOut(ev.Content)
	})

	return sink
}
