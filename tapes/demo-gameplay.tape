# tapes/demo-gameplay.tape
# README demo: gameplay loop — cursor navigation, card moves, stock draw, undo, hint.
# seed 42 / draw 1:
#   T0 → K♣   T1 → 9♦   T2 → 8♠   T3 → A♥   T4 → K♠   T5 → 3♥   T6 → 8♣

Output docs/demo/gameplay.gif

Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Framerate 15
Set PlaybackSpeed 0.9

# Launch the game
Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1.5s

# --- Move 1: A♥ (T3) → Foundation ---
# Navigate to T3 (start at T0, press Right 3 times)
Right
Sleep 200ms
Right
Sleep 200ms
Right
Sleep 200ms
# Select A♥
Enter
Sleep 400ms
# Auto-move to foundation (Space)
Type " "
Sleep 600ms

# --- Move 2: 8♠ (T2) → 9♦ (T1) ---
# Navigate back to T2 (we're at T3, go Left once)
Left
Sleep 200ms
# Select 8♠
Enter
Sleep 400ms
# Navigate left to T1
Left
Sleep 300ms
# Place on 9♦
Enter
Sleep 600ms

# --- Draw from stock ---
# Navigate to stock (T0 is leftmost; go left from T1 several times or use key 8)
Type "8"
Sleep 300ms
Type " "
Sleep 500ms
Type " "
Sleep 500ms

# --- Undo last draw ---
Type "u"
Sleep 500ms

# --- Show hint ---
Type "?"
Sleep 800ms

# Hold final state
Sleep 1s
