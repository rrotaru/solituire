# tapes/card-select.tape
# Validates: cursor highlight, drag styling, card placement rendering.

Output testdata/vhs/card-select.png
Output testdata/vhs/card-select.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/board-fresh.png

Right
Right
Right
Sleep 300ms
Screenshot testdata/vhs/cursor-on-col3.png

Enter
Sleep 300ms
Screenshot testdata/vhs/card-selected.png

Right
Right
Sleep 300ms
Enter
Sleep 300ms
Screenshot testdata/vhs/card-placed.png
