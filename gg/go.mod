module github.com/danfragoso/thdwb/gg

go 1.16

require (
	github.com/goki/freetype v1.0.5
	golang.org/x/image v0.15.0
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/hotdog => ../hotdog
)
