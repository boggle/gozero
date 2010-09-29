package main

import "fmt"
import .  "zmq"
import .  "bytes"
import .  "gonewrong"
import rt "runtime"
import "os"
import "strconv"

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
  MayPanic(OkIf(err == nil, err))
  bchan <- true
  fmt.Println("server: Bound")

  MayPanic(srv.Receive(msg, 0))
  _, err = msg.WriteTo(buf)
  if err != nil {
    fmt.Println("server: Error: ", err)
  }
  fmt.Printf("server: Received '%v'\n", buf)

  buf.Reset()
  MayPanic(srv.Receive(msg, 0))
  _, err = msg.WriteTo(buf)
  if err != nil {
    fmt.Println("server: Error: ", err)
  }

  fmt.Printf("server: Received '%v'\n", buf)
}

func Client(ctx Context, ch chan bool, tout int, addr string) {
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
  MayPanic(OkIf(err == nil, err))
  fmt.Println("client: Connected")

  fmt.Printf("client: Sending '%v'\n", buf.String())
  _, err = msg.ReadFrom(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  MayPanic(cl.Send(msg, 0))

  buf.WriteString("XXX")
  fmt.Printf("client: Sending '%v'\n", buf.String())
  _, err = msg.ReadFrom(buf)
  if err != nil {
    fmt.Println("client: Error: ", err)
  }
  MayPanic(cl.Send(msg, 0))

  // Give messages some time to actually get sent
  // (If you forget this and defer close the socket,
  // your message might not be transmitted at all)
  fmt.Printf("client: Waiting for zmq to deliver\n")
  cl.Provider().Sleep(tout)
}

func main() {
  prov := LibZmqProvider()
  ctx, err := prov.NewContext(DefaultInitArgs())
  OkIf(err != nil, err)
  defer ctx.Close()

  ch := make(chan bool)
  bchan := make(chan bool)

  if len(os.Args) != 4 {
    fmt.Println(os.Args[0], "srv|cl|all cl-timeout(ignored if srv) zmqaddr")
    os.Exit(1)
  } else {
    mode := os.Args[1]
    tout, err := strconv.Atoi(os.Args[2])
    MayPanic(err)
    addr := os.Args[3]
    switch {
    default:
      fmt.Println(os.Args[0], "srv|cl|all addr")
      os.Exit(1)
    case mode == "srv":
      go Server(ctx, ch, bchan, addr)
      <-bchan
    case mode == "cl":
      go Client(ctx, ch, tout, addr)
    case mode == "all":
      go Server(ctx, ch, bchan, addr)
      <-bchan
      go Client(ctx, ch, tout, addr)
      <-ch
    }
    fmt.Println("main: Waiting to finish")

    <-ch
  }
}
