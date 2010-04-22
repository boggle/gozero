package gozero
                       
// #include <errno.h>
// int get_errno() { return errno; }
// #include <zmq.h>
import "C"

import "unsafe"                                                         
import "runtime"
import "sync"         
import "os"   
import "syscall" 
import "strconv"    

// Constants

const (
	POLL = C.ZMQ_POLL
)

// Arguments to zmq_init
                   
type InitArgs struct {
	AppThreads 	int
	IoThreads 	int
	Flags		int
}

func DefaultInitArgs() InitArgs {  
	var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
   	if (error == nil) {
		return InitArgs{AppThreads: maxProcs, IoThreads: 1, Flags: POLL}	
	} 
	// else
	return InitArgs{AppThreads: 1, IoThreads: 1, Flags: POLL}
}
                  
// Context Wrapper

type Context struct {  
	ptr			unsafe.Pointer       	
	InitArgs

	sync.Mutex                       
}
                     
func InitDefaultContext() *Context { return InitContext(DefaultInitArgs()) }
                              
func InitContext(args InitArgs) *Context {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	
	var contextPtr unsafe.Pointer = C.zmq_init(
		C.int(args.AppThreads), 
		C.int(args.IoThreads), 
		C.int(args.Flags))
                      
    catchError(osErrorFun)	

	var context *Context = &Context{ptr: contextPtr}
	context.InitArgs = args
	runtime.SetFinalizer(context, FinalizeContext)
	return context
}                                


func FinalizeContext(context *Context) { 
	context.Lock()              
	defer context.Unlock()
	
	var contextPtr = context.ptr
	if (contextPtr != nil) {   
		context.ptr = nil		
		C.zmq_term(contextPtr) 
	}
}

// Error Handling Helpers
     
type errorFun func (int) os.Error
          
func catchError(errFun errorFun) {
	var c_errno = int(C.get_errno())
	if (c_errno != 0) {
		var error = errFun(c_errno)
		if (error == nil) {
			panic(os.NewSyscallError("Unexpected errno from zeromq", c_errno))
		} else { 
			panic(error) 
		}
	}
}

func osErrorFun(errno int) os.Error { 
	switch errno {
	case syscall.EINVAL: 
		return os.EINVAL
	default: 
  	} 
	return nil 
}
   
// {}  
