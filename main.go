package main

import "zmq"
import "runtime"
	
func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ctxCh := make(chan zmq.Context)
  // refCh := make(chan zmq.RefC)

	zmq.Thunk(func () {
		context := zmq.InitLibZmqContext(zmq.DefaultInitArgs())
    // refc := zmq.Thunk(func () { context.Terminate() }).NewRefC(1)

		srv := context.NewSocket(zmq.ZmqRep)
		defer srv.Close()
		// defer refc.Decr();

		srv.Bind("tcp://127.0.0.1:5555")

		ctxCh <- context
	  // refCh <- refc

	}).NewOSThread()

	context := <- ctxCh
  // refc    := <- refCh
	// defer refc.Incr()

	cl := context.NewSocket(zmq.ZmqReq)
	defer cl.Close()
	// defer refc.Decr()

	cl.Connect("tcp://127.0.0.1:5555")
}

