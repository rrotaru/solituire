# tapes/screens.tape
# Validates: menu screen, help overlay, pause screen layout and transitions.

Output testdata/vhs/screens.txt

Set FontSize 14
Set Width 900
Set Height 900
Set Theme "Chalk"

Type "./klondike --seed 42"
Enter
Sleep 1s
Screenshot testdata/vhs/menu-screen.png

# Navigate to Start button (menu opens on Draw Mode row; need 3 Down presses)
Down
Down
Down
Sleep 200ms

# Start game
Enter
Sleep 1s
Screenshot testdata/vhs/game-started.png

# Help overlay (H is the VHS-compatible alias for F1)
Type "H"
Sleep 500ms
Screenshot testdata/vhs/help-overlay.png

# Dismiss help (any key), then open pause
Escape
Sleep 300ms
Type "p"
Sleep 500ms
Screenshot testdata/vhs/pause-screen.png
