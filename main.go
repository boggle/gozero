package main

import .  "zmq"
import rt "runtime"
import "os"
import "syscall"
import "fmt"
	
func Server(ctx Context, refc RefC, addr string) { 
	rt.LockOSThread()
	defer rt.UnlockOSThread()
	defer refc.Decr()

	srv := ctx.NewSocket(ZmqRep)
	defer srv.Close()

	fmt.Println("About to bind server socket on ", addr)
	srv.Bind(addr)
	fmt.Println("Bound.")
	syscall.Sleep(10*1000*1000*1000)
}

func Client(ctx Context, refc RefC, addr string) { 
	rt.LockOSThread()
	defer rt.UnlockOSThread()
	defer refc.Decr()

	cl := ctx.NewSocket(ZmqRep)
	defer cl.Close()

	fmt.Println("About to connect client socket on ", addr)
	cl.Connect(addr)
	fmt.Println("Connected.")
	syscall.Sleep(10*1000*1000*1000)
}

func main() {
	ctx := InitLibZmqContext(DefaultInitArgs())
  var refc, synch = SyncOnClose(ctx, 2)

	if (len(os.Args) != 3) 	{
		fmt.Println(os.Args[0], "srv|cl|all addr")
		os.Exit(1)
	} else 	{
    mode := os.Args[1]
		addr := os.Args[2]
		switch {
			case mode == "srv":
				go Server(ctx, refc, addr)
				refc.Decr()
			case mode == "cl":
				go Client(ctx, refc, addr)
				refc.Decr()
			case mode == "all":
				go Server(ctx, refc, addr)
				go Client(ctx, refc, addr)
		}
		fmt.Println("main: Waiting to finish")
		<- synch
	} 
}

