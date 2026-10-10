# Manual live-provider recording. See README.md; run in a disposable project.
# Requires a configured OPENAI_API_KEY, micro from this checkout, Go, and VHS.
# Review the actual tool requests; do not approve unrelated actions.
# Timings depend on the provider. Adjust after a rehearsal.
Output internal/demo/chat.gif
Set Shell "bash"
Set Theme "GitHub Light"
Set FontSize 16
Set Width 1100
Set Height 720
Set TypingSpeed 35ms
Set Padding 16
Type "micro chat"
Enter
Sleep 3s
Type "Read hello.go. Change the greeting to Hello, Micro! and run go run hello.go to verify it."
Enter
Sleep 15s
# Expected: one exact edit approval, followed by one command approval.
Type "/approve"
Enter
Sleep 10s
Type "/approve"
Enter
Sleep 15s
Type "/exit"
Enter
Sleep 2s
Type "micro chat"
Enter
Sleep 4s
Type "/exit"
Enter
