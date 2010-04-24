package main

import "zmq"

func main() {
	ch := make(chan zmq.Context)
	zmq.Thunk(func () {
		context := zmq.InitLibZmqContext(zmq.DefaultInitArgs())
		ch <- context

		srv := context.NewSocket(zmq.ZmqP2P)
		defer srv.Close()
	}).GoOSThread()

	context := <- ch

	cl := context.NewSocket(zmq.ZmqP2P)
	defer cl.Close()
}

