module github.com/alfaXphoori/AgentTrack

go 1.24.0

require (
	github.com/gdamore/tcell/v2 v2.13.9
	github.com/gofrs/flock v0.13.0
	github.com/rivo/tview v0.42.0
)

require (
	github.com/creack/pty v1.1.24 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/pkoukk/tiktoken-go v0.1.8 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

retract (
	[v0.13.0, v0.14.4]
	[v0.13.0, v0.14.4]
	[v0.13.0, v0.14.4]
	[v0.1.0, v0.1.1]
	v0.1.0
)
