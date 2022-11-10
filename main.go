package main

import (
	"flag"

	"github.com/golang/glog"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			GenerateFile(gen, f)
		}
		return nil
	})

}
