package main

//go:generate go fmt ./...
//go:generate go run -mod=mod ./tools/build_locales/main.go
//go:generate go run -mod=mod ./tools/check_translations/main.go  --remove-unused
//go:generate go run -mod=mod ./ent/entc.go generate --feature ./schema
//go:generate go run -mod=mod github.com/99designs/gqlgen
