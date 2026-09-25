package v09

//go:generate go run ../../cmd/a2uigen -schemas=../../../specification/v0_9/json -pkg=v09 -stable -out=../..
//go:generate rm -rf testdata
//go:generate mkdir -p testdata/v0_9/catalogs/basic
//go:generate cp -R ../../../specification/v0_9/catalogs/basic/examples testdata/v0_9/catalogs/basic/examples
