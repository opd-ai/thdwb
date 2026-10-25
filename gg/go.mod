module github.com/danfragoso/thdwb/gg

go 1.25.0

require (
	github.com/goki/freetype v1.0.5
	golang.org/x/image v0.45.0
)

require (
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/hotdog => ../hotdog
)
