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

Type "rm -rf /tmp/project-x-vhs-demo"
Enter
Sleep 1s

Type "./px --config ./demo.config.yaml migrate"
Enter
Sleep 2s

Type "echo 'Build a tiny REST API with one health endpoint and tests' | ./px --config ./demo.config.yaml plan -"
Enter
Sleep 3s

Type "y"
Enter
Sleep 2s

Type "y"
Enter
Sleep 18s

Type "sqlite3 /tmp/project-x-vhs-demo/px.db 'select id from requirements order by created_at desc limit 1;' | xargs ./px --config ./demo.config.yaml plan --review"
Enter
Sleep 4s

Type "sqlite3 /tmp/project-x-vhs-demo/px.db 'select id from requirements order by created_at desc limit 1;' | xargs -I{} sh -c '(sleep 14; kill -INT $$) & exec ./px --config ./demo.config.yaml resume {}'"
Enter
Sleep 16s

Type "sh -c '(sleep 6; kill -INT $$) & exec ./px --config ./demo.config.yaml dashboard'"
Enter
Sleep 8s
