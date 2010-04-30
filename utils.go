package zmq

import "os"
import rt "runtime"
import "strconv"

// #include "get_errno.c"
import "C"


// ******** Thunks ********

// A simple argumentless function with no return value
type Thunk func()

// Wrap thunk in calls that lock the executing go routine to some OSThread
func (p Thunk) WithOSThread() Thunk {
  return Thunk(func() {
    rt.LockOSThread()
    defer rt.UnlockOSThread()

    p()
  })
}

// Helper for calling thunk within a separate go routine bound to a
// fixed OSThread
func (p Thunk) RunInOSThread() {
  go (p.WithOSThread())()
}

// Wrap thunk such that it sends notifi after finishing
// (May discard errors!)
func (p Thunk) Syncing(ch chan interface{}, msg interface{}) Thunk {
  return Thunk(func() {
    defer func() { ch <- msg }()
    p()
  })
}


// ******** Configuration ********

// Integer value of environment variable GOMAXPROCS if > 1, 1 otherwise
func EnvGOMAXPROCS() int {
  var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
  if error == nil && maxProcs > 1 {
    return maxProcs
  }
  return 1
}


// ******** Error Handling ********

type ErrKnow interface {
  GetError() os.Error

  OkIf(cond bool) os.Error
  ErrorIf(cond bool) os.Error
}


type ZmqErrno int

func (p ZmqErrno) String() string {
	return C.GoString(C.zmq_strerror(C.int(p)))
}

func (p ZmqErrno) IsA(err os.Errno) bool {
	return int64(p) == int64(err)
}

func (p *libZmqProvider) GetError() os.Error {
  return ZmqErrno(C.zmq_errno())
}

func (p *libZmqProvider) OkIf(cond bool) os.Error { return p.ErrorIf(!cond) }

func (p *libZmqProvider) ErrorIf(cond bool) os.Error {
  if cond {
    err := p.GetError()
    if err != nil {
      return err
    }
  }
  return nil
}

func OkIf(cond bool, error os.Error) os.Error { return ErrorIf(!cond, error) }

func ErrorIf(cond bool, error os.Error) os.Error {
  if cond {
    return error
  }
  return nil
}

func MayPanic(err os.Error) os.Error {
  if err == nil {
    return nil
  }
  panic(err)
}
