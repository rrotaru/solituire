# tapes/board-initial.tape
# Validates: full board layout, card rendering, header/footer bars.

Output testdata/vhs/board-initial.txt

Set FontSize 14
Set Width 900
Set Height 900
Set Theme "Chalk"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/board-initial.png
