# tapes/too-small.tape
# Validates: graceful degradation warning at undersized terminal (300x200).

Output testdata/vhs/too-small.txt

Set FontSize 14
Set Width 300
Set Height 200
Set Theme "Chalk"

Type "./klondike --seed 42"
Enter
Sleep 1s
Screenshot testdata/vhs/too-small-warning.png
