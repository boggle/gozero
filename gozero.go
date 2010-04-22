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
	return InitArgs{AppThreads: 1, IoThreads: 1, Flags: POLL}
}
                  
// Context Wrapper

type Context struct {  
	ptr			unsafe.Pointer       
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
                      
    catchError(func (errno int) os.Error { switch errno {
			case syscall.EINVAL: return os.EINVAL
	  	} 
		return nil })
	

	var context *Context = &Context{ptr: contextPtr}
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
                      
type ErrorFun func (int) os.Error

func catchError(errFun ErrorFun) {
	var c_errno = int(C.get_errno())
	if (c_errno != 0) {
		var error = errFun(c_errno)
		if (error == nil) {
			panic(os.NewSyscallError("Unexpected Error", c_errno))
		} else { 
			panic(error) 
		}
	}
}
