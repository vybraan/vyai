Output assets/vyai-demo.gif
Output assets/vyai-demo.mp4

Set Width 1366
Set Height 768
Set TypingSpeed 0.025

Set Margin 20
Set MarginFill "#1C2430"
Set BorderRadius 10

Type "rm -rf /tmp/vyai-demo-home && mkdir -p /tmp/vyai-demo-home && HOME=/tmp/vyai-demo-home EDITOR=vim /usr/bin/vyai"
Enter
Sleep 2s

# Chat: normal Gemini request.
Type "i"
Sleep 200ms
Type "Explain what Ken Thompson is known for in systems programming."
Enter
Sleep 12s

# Chat: demonstrate that the transcript continues as a session.
Type "Give me the practical takeaway in one sentence."
Enter
Sleep 10s

# Agent: local structured tool execution through the /agent slash command.
Type "/agent <run-bash>ls</run-bash><summary>Listed the repository root using the local agent.</summary>"
Enter
Sleep 3s

# Explore: persisted conversations, updated title, and management hints.
Tab
Sleep 1s
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# Rename preview: open title editor, then save without changing.
Type "r"
Sleep 1s
Type ":wq"
Enter
Sleep 1s

# Delete is protected by a second x press; show the confirmation notice only.
Type "x"
Sleep 1s

# Open the selected conversation again.
Enter
Sleep 1s

# Settings: editable config/prompt entries and active models.
Tab
Tab
Sleep 1s
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# Composer editor flow: open $EDITOR from the chat composer.
Tab
Sleep 500ms
Type "i"
Sleep 200ms
Type "Draft a short checklist for checking Linux disk pressure."
Ctrl+E
Sleep 1s
Type "Go"
Escape
Type ":wq"
Enter
Sleep 1s
Enter
Sleep 12s

# Finish on Settings to show the app is configurable and no longer chat-only.
Tab
Tab
Sleep 2s
