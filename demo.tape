# Project X (px) Demo
# Run: vhs demo.tape
# Produces: demo.gif

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

# Title screen
Type "# Project X (px) — AI Agent Orchestration"
Enter
Type "# Hand off a requirement. Walk away. Come back to merged PRs."
Enter
Enter
Sleep 2s

# Step 1: Show version
Type "px version"
Enter
Sleep 1s

# Step 2: Initialize
Type "px migrate"
Enter
Sleep 2s

# Step 3: Plan a requirement
Type "echo 'Build a REST API for a todo app with CRUD endpoints, SQLite storage, and tests' | px plan -"
Enter
Sleep 15s

# Step 4: Review the plan
Type "# Let's review what the planner produced..."
Enter
Sleep 1s
Type "px status"
Enter
Sleep 3s

# Step 5: Check cost
Type "px cost"
Enter
Sleep 2s

# Step 6: Show dashboard help
Type "px dashboard --help"
Enter
Sleep 2s

# Step 7: Show all commands
Type "px --help"
Enter
Sleep 3s

# Outro
Type ""
Enter
Type "# Ready to dispatch agents: px resume <req-id>"
Enter
Type "# Monitor in real-time: px dashboard"
Enter
Type "# Or in the browser: px dashboard --web"
Enter
Sleep 3s
