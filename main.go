package main

import "fmt"
import . "zmq"
import . "bytes"
import rt "runtime"
import "os"

func Server(ctx Context, ch chan bool, bchan chan bool, addr string) {
  defer func(){ ch <- true }()

  rt.LockOSThread()
  defer rt.UnlockOSThread()

  srv := ctx.NewSocket(ZmqP2P)
  defer srv.Close()

  fmt.Println("server: About to bind server socket on ", addr)
  srv.Bind(addr)
  bchan <- true
  fmt.Println("server: Bound")

	msg := srv.Provider().NewMessage()
	buf := NewBuffer(make([]byte, 2))
	defer msg.Close()

	srv.Receive(msg, 0)
	var _, err = msg.GetData(buf)	
	if (err != nil) { fmt.Println(err) }
	fmt.Println("server: Received '", buf.String(), "'")

	srv.Receive(msg, 0)
	buf.Reset()
	_, err = msg.GetData(buf)	
	if (err != nil) { fmt.Println(err) }
	fmt.Println("server: Received '", buf.String(), "'")
}

func Client(ctx Context, ch chan bool, addr string) {
  defer func(){ ch <- true }()

  rt.LockOSThread()
  defer rt.UnlockOSThread()

  cl := ctx.NewSocket(ZmqP2P)
  defer cl.Close()

  fmt.Println("client: About to connect client socket on ", addr)
  cl.Connect(addr)
  fmt.Println("client: Connected")

	buf := NewBufferString("!PING!PING!PING!")
	msg := cl.Provider().NewMessage()
	defer msg.Close()

	var _, err = msg.SetData(buf)
	if (err != nil) { fmt.Println("client: ", err) }
	fmt.Println("client: Sending '", buf.String(), "'")
	cl.Send(msg, 0)

	buf.Reset()
	buf.WriteString("XXX")
	 _, err = msg.SetData(buf)
	if (err != nil) { fmt.Println("client: ", err) }
	fmt.Println("client: Sending '", buf.String(), "'")
	cl.Send(msg, 0)
}

func main() {
  ctx   := LibZmqProvider().NewContext(DefaultInitArgs())
  defer ctx.Close()
	ch    := make(chan bool)
	bchan := make(chan bool)

  if len(os.Args) != 3 {
    fmt.Println(os.Args[0], "srv|cl|all addr")
    os.Exit(1)
  } else {
    mode := os.Args[1]
    addr := os.Args[2]
    switch {
    case mode == "srv":
      Server(ctx, ch, bchan, addr)
			<- bchan
    case mode == "cl":
      Client(ctx, ch, addr)
    case mode == "all":
      go Server(ctx, ch, bchan, addr)
			<- bchan
      go Client(ctx, ch, addr)
	    <- ch
    }
    fmt.Println("main: Waiting to finish")
    <- ch
  }
}
