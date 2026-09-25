package v10

//go:generate go run ../../cmd/a2uigen -schemas=../../../specification/v1_0/json -catalog=../../../catalogs/basic/v1/catalog.json -pkg=v10 -out=../..
//go:generate rm -rf testdata
//go:generate mkdir -p testdata/v1_0/catalogs/basic
//go:generate cp -R ../../../catalogs/basic/v1/examples testdata/v1_0/catalogs/basic/examples
