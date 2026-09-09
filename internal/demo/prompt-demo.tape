# VHS tape: the hero demo — generate a system from a prompt and talk to it.
# Record with: ANTHROPIC_API_KEY=... vhs internal/demo/prompt-demo.tape
#
# MANUAL RECORDING ONLY: generation time depends on the live provider, so the
# sleeps below are generous estimates — expect to tune them (or trim the video)
# per recording. Record in an empty scratch directory.

Output internal/demo/prompt-demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1100
Set Height 700
Set TypingSpeed 40ms
Set Padding 12

Type `micro run --prompt "a task management system with categories" --provider anthropic`
Enter
# Wait for the design proposal (Services: task / project ... Generate? [Y/n])
Sleep 25s

Type "y"
Enter
# Wait for codegen + compile + start + console banner
Sleep 60s

Type "Create a project called Launch, then add three tasks to it"
Enter
# Tool calls stream: project_Project_Create, task_Task_Create x3
Sleep 30s

Type "I need to track shipping. Create a shipment for order 123 to London."
Enter
# The mid-conversation service generation moment — the money shot.
Sleep 60s

Ctrl+C
Sleep 2s
