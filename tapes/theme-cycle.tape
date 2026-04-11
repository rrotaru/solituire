# tapes/theme-cycle.tape
# Validates: each theme renders correctly (colors, contrast, borders).
# Only produces a .txt output; per-theme screenshots are more useful than a final frame.

Output testdata/vhs/theme-cycle.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/theme-classic.png

Type "t"
Sleep 500ms
Screenshot testdata/vhs/theme-dracula.png

Type "t"
Sleep 500ms
Screenshot testdata/vhs/theme-solarized.png
