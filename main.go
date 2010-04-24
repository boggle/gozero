package main

import "zmq"

func main() {
	context := (<- zmq.WithOSThread(func (thr *zmq.GoThread, ch chan interface{}) {
		context := zmq.InitLibZmqContext(zmq.DefaultInitArgs())
		ch <- context

	})).(zmq.Context)
	defer context.Close()
}

