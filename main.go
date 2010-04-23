package main

import "gozero"

func main() {
	var thr = gozero.NewGoThread()
	defer thr.Finish()

	gozero.InitDefaultContext()
}
