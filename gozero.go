package gozero
                       
// #include <zmq.h>
// #include <stdlib.h>
// #include "get_errno.h"
import "C"

import "unsafe"                                                         
import "sync"         
import "os"   
import "runtime"   
import "syscall" 
import "strconv"     

// Constants

const (
	ZmqPoll 		  = C.ZMQ_POLL
  ZmqP2P  		  = C.ZMQ_P2P
  ZmqPub  		  = C.ZMQ_PUB
  ZmqSub  		  = C.ZMQ_SUB
  ZmqReq  		  = C.ZMQ_REQ
  ZmqRep  		  = C.ZMQ_REP
  ZmqUpstream   = C.ZMQ_UPSTREAM
	ZmqDownstream = C.ZMQ_DOWNSTREAM
)

// Arguments to zmq_init
                   
type InitArgs struct {
	AppThreads 	int
	IoThreads 	int
	Flags				int
}

func DefaultInitArgs() InitArgs {  
	var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
   	if (error == nil) {
		return InitArgs{AppThreads: maxProcs, IoThreads: 1, Flags: ZmqPoll}	
	} 
	// else
	return InitArgs{AppThreads: 1, IoThreads: 1, Flags: ZmqPoll}
}
                  
// Context Wrapper
type Context struct {  
	ptr			unsafe.Pointer       	
	InitArgs

	lock		sync.Mutex                       
}
                     
func InitDefaultContext() *Context { 
	return InitContext(DefaultInitArgs()) 
}
                              
func InitContext(args InitArgs) *Context {
	
	var contextPtr unsafe.Pointer = C.zmq_init(
		C.int(args.AppThreads), 
		C.int(args.IoThreads), 
		C.int(args.Flags))
                      
    CatchError(zmqErrnoFun)	

	var context *Context = &Context{ptr: contextPtr}
	context.InitArgs = args
	runtime.SetFinalizer(context, finalizeContext)
	return context
}                                

func (p *Context) Terminate() {
	p.lock.Lock()
	defer p.lock.Unlock()

	var contextPtr = p.ptr
	if (contextPtr != nil) {
		p.ptr = nil
		C.zmq_term(contextPtr) 
	}
}

func finalizeContext(context *Context) {
	go func() { 
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	
		context.Terminate()
	}()
}

func zmqErrnoFun(errno int) os.Error {
  switch errno {
  case syscall.EINVAL:
    return os.EINVAL
  default:
  }
  return nil
}

type Socket struct {
	ptr		unsafe.Pointer
	thr		*GoThread
}

func (p *Context) NewSocket(thr *GoThread, socketType int) *Socket {
	var socket *Socket = new(Socket)
	socket.thr 				 = thr
	thr.OnFinish(socket)

	socket.ptr = C.zmq_socket(p.ptr, C.int(socketType))
	return socket
}

var AlreadyClosed = os.NewError("Socket already closed")

func (p *Socket) Close() int {
	var ptr = p.ptr
	p.ptr = nil
	MayPanic(p == nil, AlreadyClosed)
	return int(C.zmq_close(ptr))
}

func (p *Socket) OnOSThreadLock(thr *GoThread) {
}

func (p *Socket) OnOSThreadUnlock(thr *GoThread) {
	p.Close()
}
