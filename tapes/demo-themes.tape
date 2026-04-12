# tapes/demo-themes.tape
# README demo: all five built-in themes cycling on a live game board.

Output docs/demo/themes.gif

Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Framerate 12
Set PlaybackSpeed 0.9

# Launch the game
Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1.5s

# Classic (default — already showing)
Sleep 1.2s

# Dracula
Type "t"
Sleep 1.2s

# Solarized Dark
Type "t"
Sleep 1.2s

# Solarized Light
Type "t"
Sleep 1.2s

# Nord
Type "t"
Sleep 1.2s

# Cycle back to Classic
Type "t"
Sleep 1s
