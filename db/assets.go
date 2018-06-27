// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(http.Dir("./sql"), vfsgen.Options{
		PackageName:  "opbeansdb",
		VariableName: "SQL",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
