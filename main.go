package main

import "zmq"

func main() {
	ctxCh := make(chan zmq.Context)
  refCh := make(chan zmq.RefC)

	zmq.Thunk(func () {
		context := zmq.InitLibZmqContext(zmq.DefaultInitArgs())
    refc := zmq.Thunk(func () { context.Terminate() }).NewRefC(2)
		defer refc.Decr();

		ctxCh <- context
	  refCh <- refc

		srv := context.NewSocket(zmq.ZmqP2P)
		defer srv.Close()

	}).GoOSThread()

	context := <- ctxCh
  refc    := <- refCh
	defer refc.Decr()

	cl := context.NewSocket(zmq.ZmqP2P)
	defer cl.Close()

}

