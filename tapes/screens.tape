# tapes/screens.tape
# Validates: menu screen, help overlay, pause screen layout and transitions.

Output testdata/vhs/screens.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42"
Enter
Sleep 1s
Screenshot testdata/vhs/menu-screen.png

# Start game
Enter
Sleep 1s
Screenshot testdata/vhs/game-started.png

# Help overlay
Type "F1"
Sleep 500ms
Screenshot testdata/vhs/help-overlay.png

# Dismiss help, open pause
Escape
Sleep 300ms
Type "p"
Sleep 500ms
Screenshot testdata/vhs/pause-screen.png
