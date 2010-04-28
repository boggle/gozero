package main

import "fmt"
import . "zmq"
import . "bytes"
import rt "runtime"
import "os"
// import "syscall"

func Server(ctx Context, ch chan bool, bchan chan bool, addr string) {
  defer func() { ch <- true }()

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
  if err != nil {
    fmt.Println("server: Error: ", err)
  }
  fmt.Printf("server: Received '%v'\n", buf)

  srv.Receive(msg, 0)
  buf.Reset()
  _, err = msg.GetData(buf)
  if err != nil {
    fmt.Println("server: Error: ", err)
  }
  fmt.Printf("server: Received '%v'\n", buf)
}

func Client(ctx Context, ch chan bool, addr string) {
  defer func() { ch <- true }()

  rt.LockOSThread()
  defer rt.UnlockOSThread()

  cl := ctx.NewSocket(ZmqP2P)
  defer cl.Close()

  fmt.Println("client: About to connect client socket on ", addr)
  cl.Connect(addr)
  fmt.Println("client: Connected")

  buf := NewBufferString("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
  msg := cl.Provider().NewMessage()
  defer msg.Close()

  fmt.Printf("client: Sending '%v'\n", buf.String())
  var _, err = msg.SetData(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  cl.Send(msg, 0)

  buf.WriteString("XXX")
  fmt.Printf("client: Sending '%v'\n", buf.String())
  _, err = msg.SetData(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  cl.Send(msg, 0)
}

func main() {
  ctx := LibZmqProvider().NewContext(DefaultInitArgs())
  defer ctx.Close()
  ctx2 := LibZmqProvider().NewContext(DefaultInitArgs())
  defer ctx2.Close()

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
      go Server(ctx2, ch, bchan, addr)
      <-bchan
      go Client(ctx, ch, addr)
      <-ch
    }
    fmt.Println("main: Waiting to finish")

    <-ch
  }
}
