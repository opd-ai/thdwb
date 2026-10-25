module github.com/danfragoso/thdwb/sauce

go 1.25.0

require (
	github.com/danfragoso/thdwb/assets v0.0.0-20210612223625-beb2b4a85bbb
	github.com/danfragoso/thdwb/pages v0.0.0-20210411220950-eaec6f13eccb
	github.com/stretchr/testify v1.11.1
	golang.org/x/net v0.57.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/hotdog => ../hotdog
	github.com/danfragoso/thdwb/pages => ../pages
)
