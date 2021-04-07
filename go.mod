module github.com/terra-farm/terraform-provider-virtualbox

go 1.14

// github.com/congop/execstub => ../../congop/execstub/
// replace github.com/terra-farm/go-virtualbox => ../go-virtualbox

replace github.com/terra-farm/go-virtualbox => github.com/congop/go-virtualbox v0.0.0-20210405132148-940b224122f0

require (
	github.com/ajvb/kala v0.3.3
	github.com/congop/execstub v0.0.0-20210402081209-aaa19f24dc75
	github.com/dustin/go-humanize v1.0.0
	github.com/godoctor/godoctor v0.0.0-20181123222458-69df17f3a6f6
	github.com/google/uuid v1.1.1
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/terraform-plugin-sdk v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.6.1
	github.com/terra-farm/go-virtualbox v0.0.0
	golang.org/x/exp v0.0.0-20190510132918-efd6b22b2522
)

//github.com/terra-farm/go-virtualbox v0.0.
