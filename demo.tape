# Project X (px) fallback planning demo
# Run: vhs demo.tape
# Produces: demo.gif

Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1500
Set Height 900
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

Type "cd /Users/mncedimini/Sites/hackathon"
Enter
Sleep 1s

Type "echo 'Build a tiny REST API with one health endpoint and tests' | ./px --config ./px.config.yaml plan -"
Enter
Sleep 3s

Type "y"
Enter
Sleep 2s

Type "y"
Enter
Sleep 18s

Type "sqlite3 /Users/mncedimini/.px/px.db 'select id from requirements order by created_at desc limit 1;' | xargs ./px --config ./px.config.yaml plan --review"
Enter
Sleep 4s
