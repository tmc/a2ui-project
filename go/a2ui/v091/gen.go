package v091

//go:generate go run ../../cmd/a2uigen -schemas=../../../specification/v0_9_1/json -pkg=v091 -out=../..
//go:generate rm -rf testdata
//go:generate mkdir -p testdata/v0_9_1/catalogs/basic
//go:generate cp -R ../../../specification/v0_9_1/catalogs/basic/examples testdata/v0_9_1/catalogs/basic/examples
