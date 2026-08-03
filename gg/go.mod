module github.com/danfragoso/thdwb/gg

go 1.16

require (
	github.com/goki/freetype v0.0.0-20181231101311-fa8a33aabaff
	golang.org/x/image v0.0.0-20210220032944-ac19c3e999fb
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/hotdog => ../hotdog
)
