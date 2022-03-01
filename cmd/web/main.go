package main

import (
	"context"
	"log"

	"github.com/kubesphere/kubeeye/web"
)

func main() {
	log.Fatalln(web.RunWebService(context.Background()))
}
