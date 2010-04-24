package zmq
                       
// #include <zmq.h>
// #include <stdlib.h>
// #include "get_errno.h"
import "C"

import "unsafe"                                                         
import "sync"         
import "os"   
import "runtime"   
import "strconv"     



// ******** Global ZMQ Constants ********

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



// ********* Contexts and InitArgs **********

// ZMQ Context type
type Context interface {
	Closeable

	NewSocket(thr *GoThread, socketType int) Socket
	Terminate()
}

// libzmq context wrapper
type lzmqContext struct {  
	Context

	ptr				unsafe.Pointer       	
	initArgs	InitArgs

	lock			sync.RWMutex                       
}
// Arguments to zmq_init
type InitArgs struct {
	AppThreads 	int
	IoThreads 	int
	Flags				int
}

// Sensible default init args
// AppThreads = Max(GOMAXPROCS, 1), IoThreads = 1, Flags = ZmqPoll
func DefaultInitArgs() InitArgs {  
	var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
   	if (error == nil) {
		return InitArgs{AppThreads: maxProcs, IoThreads: 1, Flags: ZmqPoll}	
	} 
	// else
	return InitArgs{AppThreads: 1, IoThreads: 1, Flags: ZmqPoll}
}
                     
// Setup a program-wide thread-safe zmq context object
// Expects to be run in a separate go routine safely locked to an OS Thread
func InitLibZmqContext(args InitArgs) Context {
	var contextPtr unsafe.Pointer = C.zmq_init(
		C.int(args.AppThreads), 
		C.int(args.IoThreads), 
		C.int(args.Flags))
                      
  CondCatchError(contextPtr == nil, libZmqErrnoFun)	

	lzmqContext := &lzmqContext{ptr: contextPtr, initArgs: args}
	runtime.SetFinalizer(lzmqContext, finalizeContext)
	return lzmqContext
}                                

// Returns InitArgs
func (context *lzmqContext) GetInitArgs() InitArgs {
	return context.initArgs
}

// If context is open and openState is true, calls thunk and returns true.
// If context has been closed and openState is false, calls thunk and returns true.
// Otherwise returns false.
func (context *lzmqContext) CondDo(openState bool, thunk func()) bool {
	context.lock.RLock()
	defer context.lock.RUnlock()

	if ((context.ptr != nil) == openState) { thunk(); return true; }
	return false;
}

// Finalizer for lzmqContexts.  Calls Terminate().
func finalizeContext(context *lzmqContext) { context.Terminate() }

// Same as Terminate()
func (p *lzmqContext) Close() { p.Terminate() }

// Call zmq_term on underlying context pointer.
// Needs to run in separate GoRoutine to safely lock the OS Thread
// and synchronize via channel to know when we're done
func (p *lzmqContext) Terminate() {
	// Wait till zmq_term has finished
	<- WithOSThread(func (thr *GoThread, ch chan interface{}) {
		p.lock.Lock()
		defer p.lock.Unlock()

		var contextPtr = p.ptr
		if (contextPtr != nil) {
			p.ptr = nil
			ret := int(C.zmq_term(contextPtr))
			CondCatchError(ret == -1, libZmqErrnoFun)
		}

		ch <- nil
	})
}


// ******** Sockets ********

// ZMQ Socket type
type Socket interface{
	Closeable
	OSThreadBound
}

// libzmq socket wrapper
type lzmqSocket struct {
	Socket

	ptr		unsafe.Pointer
	thr		*GoThread
}

func (p *lzmqContext) NewSocket(thr *GoThread, socketType int) Socket {
	thr.EnsureOSThreadBound()
	p.lock.RLock()
	defer p.lock.RUnlock()

	socket    := new(lzmqSocket)
	socket.thr = thr
	socket.ptr = C.zmq_socket(p.ptr, C.int(socketType))
	CondCatchError(socket.ptr == nil, libZmqErrnoFun)
	thr.Register(socket)

	return socket
}

// If Socket is open and openState is true, calls thunk and returns true.
// If Socket has been closed and openState is false, calls thunk and 
// returns true.
// Otherwise returns false.
// Expects the socket's interal GoThread to still be locked onto an OSThread.
func (p *lzmqSocket) CondDo(openState bool, thunk func()) bool {
	p.thr.EnsureOSThreadBound()
	
	if ((p.ptr != nil) == openState) { thunk(); return true }
	return false
}

// Close this socket.
// Expects the socket's interal GoThread to still be locked onto an OSThread.
func (p *lzmqSocket) Close() {
	p.thr.EnsureOSThreadBound()

	socketPtr := p.ptr
	CondPanic(socketPtr == nil, errAlreadyClosed)

	p.ptr = nil
	CondCatchError(int(C.zmq_close(socketPtr)) == -1, libZmqErrnoFun)
}

func (p *lzmqSocket) OnOSThreadLock(thr *GoThread) {
}

func (p *lzmqSocket) OnOSThreadUnlock(thr *GoThread) {
	socketPtr := p.ptr
	if (socketPtr != nil) {
		p.ptr = nil
		CondCatchError(int(C.zmq_close(socketPtr)) == -1, libZmqErrnoFun)
	}
}




// ******** LibZmq Error Handling *******

var errEMTHREAD os.Error = os.NewError("ZMQ: EMTHREAD")
var errEFSM os.Error = os.NewError("ZMQ: EFSM")
var errENOCOMPATPROTO os.Error = os.NewError("ZMQ: ENOCOMPATPROTO")

// Default ErrnoFun used for libzmq syscalls
func libZmqErrnoFun(errno os.Errno) os.Error {
  switch errno {
  case os.Errno(C.EMTHREAD):
		return errEMTHREAD	
  case os.Errno(C.EFSM):
		return errEFSM
  case os.Errno(C.ENOCOMPATPROTO):
		return errENOCOMPATPROTO
  default:
  }
  return nil
}

// {}

