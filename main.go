package main

import "fmt"
import . "zmq"
import . "bytes"
import rt "runtime"
import "os"
// import "syscall"

func Server(ctx Context, ch chan bool, bchan chan bool, addr string) {
  rt.LockOSThread()
  defer rt.UnlockOSThread()

  defer func() { ch <- true }()

  srv, err := ctx.NewSocket(ZmqP2P)
  OkIf(err != nil, err)
  defer srv.Close()

  msg := srv.Provider().NewMessage()
  buf := NewBuffer(make([]byte, 2))
  defer msg.Close()

  fmt.Println("server: About to bind server socket on ", addr)
  err = srv.Bind(addr)
  OkIf(err != nil, err)
  bchan <- true
  fmt.Println("server: Bound")

  MayPanic(srv.Receive(msg, 0))
  _, err = msg.GetData(buf)
  if err != nil {
    fmt.Println("server: Error: ", err)
  }
  fmt.Printf("server: Received '%v'\n", buf)

  buf.Reset()
  MayPanic(srv.Receive(msg, 0))
  _, err = msg.GetData(buf)
  if err != nil {
    fmt.Println("server: Error: ", err)
  }

  fmt.Printf("server: Received '%v'\n", buf)
}

func Client(ctx Context, ch chan bool, addr string) {
  rt.LockOSThread()
  defer rt.UnlockOSThread()

  defer func() { ch <- true }()

  cl, err := ctx.NewSocket(ZmqP2P)
  OkIf(err != nil, err)
  defer cl.Close()

  buf := NewBufferString("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
  msg := cl.Provider().NewMessage()
  defer msg.Close()

  fmt.Println("client: About to connect client socket on ", addr)
  err = cl.Connect(addr)
  OkIf(err != nil, err)
  fmt.Println("client: Connected")

  fmt.Printf("client: Sending '%v'\n", buf.String())
  _, err = msg.SetData(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  MayPanic(cl.Send(msg, 0))

  buf.WriteString("XXX")
  fmt.Printf("client: Sending '%v'\n", buf.String())
  _, err = msg.SetData(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  MayPanic(cl.Send(msg, 0))

	// Give messages some time to actually get sent
	// (If you forget this and defer close the socket,
  // your message might not be transmitted at all)
  fmt.Printf("client: Waiting for zmq to deliver\n", buf.String())
	cl.Provider().Sleep(2)
}

func main() {
  prov := LibZmqProvider()
  ctx, err := prov.NewContext(DefaultInitArgs())
  OkIf(err != nil, err)
  defer ctx.Close()

  ch := make(chan bool)
  bchan := make(chan bool)

  if len(os.Args) != 3 {
    fmt.Println(os.Args[0], "srv|cl|all addr")
    os.Exit(1)
  } else {
    mode := os.Args[1]
    addr := os.Args[2]
    switch {
    default:
      fmt.Println(os.Args[0], "srv|cl|all addr")
      os.Exit(1)
    case mode == "srv":
      go Server(ctx, ch, bchan, addr)
      <-bchan
      // syscall.Sleep(10 * 1000 * 1000 * 1000)
    case mode == "cl":
      go Client(ctx, ch, addr)
    case mode == "all":
      go Server(ctx, ch, bchan, addr)
      <-bchan
      go Client(ctx, ch, addr)
      <-ch
    }
    fmt.Println("main: Waiting to finish")

    <-ch
  }
}
