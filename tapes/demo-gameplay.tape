# tapes/demo-gameplay.tape
# README demo: gameplay loop — cursor navigation, card moves, stock draw, undo/redo, hint.
# seed 42 / draw 1 initial layout:
#   T0: K♣  T1: 9♦  T2: 8♠  T3: A♥  T4: K♠  T5: 3♥  T6: 8♣

Output docs/demo/gameplay.gif

Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Framerate 15
Set PlaybackSpeed 0.9

# Launch the game (--draw explicit skips the menu)
Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1.5s

# --- Move 1: 8♠ (T2) → 9♦ (T1) ---
# Cursor starts at T0; navigate Right to T2
Right
Sleep 200ms
Right
Sleep 200ms
# Select 8♠
Enter
Sleep 400ms
# Navigate left to T1 (9♦) and place
Left
Sleep 300ms
Enter
Sleep 700ms

# --- Move 2: A♥ (T3) → Foundation using 'f' ---
# From T1, go Right twice to reach T3
Right
Sleep 200ms
Right
Sleep 200ms
# Auto-move A♥ straight to foundation ('f' works without picking up first)
Type "f"
Sleep 700ms

# --- Draw two cards from stock (Space always flips stock) ---
Space
Sleep 500ms
Space
Sleep 500ms

# --- Undo last draw (Ctrl+Z) ---
Ctrl+Z
Sleep 600ms

# --- Show hint ---
Type "?"
Sleep 1s

# Hold final state
Sleep 0.5s
