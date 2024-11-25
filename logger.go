package main

import (
	"log"
	"os"
)

var mylog = log.New(os.Stderr, "ProxyServer: ", log.LstdFlags|log.Lshortfile)
