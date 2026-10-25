module github.com/danfragoso/thdwb/browser

go 1.25.0

require (
	github.com/danfragoso/thdwb/assets v0.0.0
	github.com/danfragoso/thdwb/hotdog v0.0.0
	github.com/danfragoso/thdwb/mustard v0.0.0
	github.com/danfragoso/thdwb/profiler v0.0.0
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/hotdog => ../hotdog
	github.com/danfragoso/thdwb/mustard => ../mustard
	github.com/danfragoso/thdwb/profiler => ../profiler
)
