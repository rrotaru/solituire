# tapes/card-select.tape
# Validates: cursor highlight, drag styling, card placement rendering.
# Valid move with seed 42: 8♠ (T2, black) → 9♦ (T1, red) — alternating color, rank 8 on 9.

Output testdata/vhs/card-select.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/board-fresh.png

# Navigate to T2 (8♠): cursor starts at T0, two Rights → T1 → T2
Right
Right
Sleep 300ms
Screenshot testdata/vhs/cursor-on-t2.png

# Select 8♠ at T2
Enter
Sleep 300ms
Screenshot testdata/vhs/card-selected.png

# Navigate left to T1 (9♦) — the valid destination
Left
Sleep 300ms
Enter
Sleep 300ms
Screenshot testdata/vhs/card-placed.png
