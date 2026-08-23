# VHS tape: the no-key quick start — scaffold → run → call.
# Record with: vhs internal/demo/first-run.tape  (see internal/demo/README.md)
# Requires: vhs, ttyd, ffmpeg, and the micro CLI on PATH. No provider key needed.

Output internal/demo/first-run.gif

Set Shell "bash"
Set FontSize 15
Set Width 1100
Set Height 620
Set TypingSpeed 40ms
Set Padding 12

Type "micro new helloworld"
Enter
Sleep 4s

Type "cd helloworld"
Enter
Sleep 500ms

# Loopback bind → auth off → the curl below needs no token.
Type "micro run -d &"
Enter
Sleep 8s

Type `curl -s -X POST localhost:8080/api/helloworld/Helloworld.Call -H 'Content-Type: application/json' -d '{"name":"World"}'`
Enter
Sleep 4s

Hide
Type "kill %1"
Enter
Sleep 1s
Show
