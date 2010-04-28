package zmq

// import "fmt"
import . "bytes"
import "io"
import "os"
import "unsafe"
import . "unsafe/coffer"

// #include <zmq.h>
// #include <stdlib.h>
// #include "get_errno.h"
import "C"


// ******** Global ZMQ Constants ********

const (
  ZmqPoll       = C.ZMQ_POLL
  ZmqP2P        = C.ZMQ_P2P
  ZmqPub        = C.ZMQ_PUB
  ZmqSub        = C.ZMQ_SUB
  ZmqReq        = C.ZMQ_REQ
  ZmqRep        = C.ZMQ_REP
  ZmqUpstream   = C.ZMQ_UPSTREAM
  ZmqDownstream = C.ZMQ_DOWNSTREAM
  ZmqNoBlock    = C.ZMQ_NOBLOCK
)


// ******** ZMQ Interfaces ********

type Provider interface {
  NewContext(initArgs InitArgs) Context
  NewMessage() Message
}

type Provided interface {
  Provider() Provider
}

// Arguments to New Context
type InitArgs struct {
  AppThreads int
  IoThreads  int
  Flags      int
}

// Sensible default init args
// AppThreads = EnvGOMAXPROCS(), IoThreads = 1, Flags = ZmqPoll
func DefaultInitArgs() InitArgs {
  return InitArgs{AppThreads: EnvGOMAXPROCS(), IoThreads: 1, Flags: ZmqPoll}
}

// Context interface
//
// Contexts are always global thread-safe objects
type Context interface {
  io.Closer
  Provided

  NewSocket(socketType int) Socket
  Terminate()
}

// Message interface
//
// Messages may only be used reliably with sockets from the same provider
type Message interface {
  io.Closer
  Provided

  // Empty()
  // Allocate(cap int)

  GetData(buf *Buffer) (n int, err os.Error)
  SetData(buf *Buffer) (n int, err os.Error)

  Size() int
}

// Socket interface
//
// Sockets are typically thread-bound
type Socket interface {
  io.Closer
  Provided

  Bind(address string)
  Connect(address string)

  Receive(msg Message, flags int) bool
  Send(msg Message, flags int) bool

  // TODO
  // Poll?
}


// ******** lzmq: Provider ********

// libzmq provider impl
func LibZmqProvider() Provider { return _libZmqProvider }

type libZmqProvider struct{}

var _libZmqProvider = new(libZmqProvider)


// ******** lzmq: Context ********

// libzmq context wrapper
type lzmqContext uintptr

// Creates a zmq context and returns it.
//
// Don't forget to set GOMAunsafe.ROCS appropriately when working with libzmq.
//
// Contexts are finalized by the GC unless they are manually destructed
// by calling Terminate() beforehand.  Applications need to arrange
// that no socket is used or even closed after the owning context has
// been destructed.  This requires to have at least one running go routine
// with a live referene to the context.
func (p libZmqProvider) NewContext(args InitArgs) Context {
  contextPtr := C.zmq_init(
    C.int(args.AppThreads),
    C.int(args.IoThreads),
    C.int(args.Flags))

  CondCatchError(contextPtr == nil, libZmqErrnoFun)

  lzmqContext := lzmqContext(contextPtr)
  return lzmqContext
}


func (p *libZmqProvider) NewMessage() Message {
  msg := new(lzmqMessage)
  // msg.empty()
  return msg
}

func (p lzmqContext) Provider() Provider { return LibZmqProvider() }

// Calls Terminate()
func (p lzmqContext) Close() os.Error {
  p.Terminate()
  return nil
}

// Calls zmq_term on underlying context pointer
//
// Only call once
func (p lzmqContext) Terminate() {
  ch := make(chan interface{})
  ptr := unsafe.Pointer(p)
  if ptr != nil {
    // Needs to run in separate go routine to safely lock the OS Thread
    // and synchronize via channel to know when we're done
    Thunk(func() {
      CondCatchError(int(C.zmq_term(ptr)) == -1, libZmqErrnoFun)
      ch <- nil
    }).RunInOSThread()
    // Wait for completion
    <-ch
  }
}


// ******** lzmq: Messages ********

type lzmqMessageHolder interface {
  Message

  getLzmqMessage() *lzmqMessage
}

type lzmqMessage C.zmq_msg_t

func (p *lzmqMessage) empty() {
  CondCatchError(C.zmq_msg_init(p.ptr()) == -1, libZmqErrnoFun)
}

func (p *lzmqMessage) allocate(length int) {
  CondCatchError(C.zmq_msg_init_size(p.ptr(), C.size_t(length)) == -1, libZmqErrnoFun)
}

func (p *lzmqMessage) Provider() Provider { return LibZmqProvider() }

func (p *lzmqMessage) Size() int {
  return int(C.zmq_msg_size(p.ptr()))
}

func (p *lzmqMessage) GetData(buf *Buffer) (n int, err os.Error) {
  n = p.Size()
  start := p.data()
  var coffr *Coffer
  coffr, err = NewCoffer(start, n)
  if err != nil {
    return n, err
  }

  var n64 int64
  n64, err = buf.ReadFrom(coffr)
  if n64 == int64(n) && err == os.EOF {
    err = nil
  }
  return int(n64), err
}

func (p *lzmqMessage) SetData(buf *Buffer) (n int, err os.Error) {
  n = buf.Len()
  if n <= 0 {
    return 0, os.EOF
  }
  p.allocate(n)

  start := p.data()
  var coffr *Coffer
  coffr, err = NewCoffer(start, n)
  if err != nil {
    return n, err
  }

  var n64 int64
  n64, err = buf.WriteTo(coffr)
  if n64 == int64(n) && err == os.EOF {
    err = nil
  }
  return int(n64), err
}

func (p *lzmqMessage) ptr() *C.zmq_msg_t {
  return (*C.zmq_msg_t)(unsafe.Pointer(p))
}

func (p *lzmqMessage) data() uintptr {
  return uintptr(C.zmq_msg_data(p.ptr()))
}

func (p *lzmqMessage) getLzmqMessage() *lzmqMessage {
  return p
}

func (p *lzmqMessage) Close() os.Error {
  if C.zmq_msg_close(p.ptr()) == -1 {
    return FetchError(errno(), libZmqErrnoFun)
  }
  return nil
}


// ******** lzmq: Sockets ********

// libzmq socket wrapper
type lzmqSocket uintptr

// Creates a new Socket with the given socketType
//
// Sockets only must be used from a fixed OSThread. This may be achieved
// by conveniently using Thunk.NewOSThread() or by calling runtime.LockOSThread()
func (p lzmqContext) NewSocket(socketType int) Socket {
  ptr := unsafe.Pointer(C.zmq_socket(unsafe.Pointer(p), C.int(socketType)))
  CondCatchError(ptr == nil, libZmqErrnoFun)
  return lzmqSocket(ptr)
}

func (p lzmqSocket) Provider() Provider { return LibZmqProvider() }

// Bind server socket
func (p lzmqSocket) Bind(address string) {
  ptr := unsafe.Pointer(p)
  // apparantly freed by zmq
  c_addr := C.CString(address)
  CondCatchError(C.zmq_bind(ptr, c_addr) == -1, libZmqErrnoFun)
}

// Connect client socket
func (p lzmqSocket) Connect(address string) {
  ptr := unsafe.Pointer(p)
  // apparantly freed by zmq
  c_addr := C.CString(address)
  CondCatchError(C.zmq_connect(ptr, c_addr) == -1, libZmqErrnoFun)
}

func (p lzmqSocket) Receive(msg Message, flags int) bool {
  lzmqMsg := msg2lzmqMessage(msg)
  lzmqMsg.empty()
  ret := int(C.zmq_recv(unsafe.Pointer(p), lzmqMsg.ptr(), C.int(flags)))
  // fmt.Println("recv", msg)
  return msgCheckErrno(ret, flags)
}

func (p lzmqSocket) Send(msg Message, flags int) bool {
  lzmqMsg := msg2lzmqMessage(msg)
  // fmt.Println("send", msg)
  ret := int(C.zmq_send(unsafe.Pointer(p), lzmqMsg.ptr(), C.int(flags)))
  return msgCheckErrno(ret, flags)
}

func msg2lzmqMessage(msg Message) *lzmqMessage {
  var lzmqMsgHolder, err = msg.(lzmqMessageHolder)
  if !err {
    panic(os.EINVAL)
  }
  lzmqMsg := lzmqMsgHolder.getLzmqMessage()
  return lzmqMsg
}

func msgCheckErrno(ret int, flags int) bool {
  if ret == -1 {
    c_errno := errno()
    if (flags&ZmqNoBlock != 0) && (c_errno == C.EAGAIN) {
      // Try again
      return false
    }
    // Regular fail
    CatchErrno(c_errno, libZmqErrnoFun)
    // Unreachable
    panic(os.EINVAL)
  }
  // Message was processed
  return true
}

// Closes this socket
//
// Expects the executing go routine to still be locked onto an OSThread.
// May be called only once
func (p lzmqSocket) Close() os.Error {
  if int(C.zmq_close(unsafe.Pointer(p))) == -1 {
    return FetchError(errno(), libZmqErrnoFun)
  }
  return nil
}


// ******** LibZmq Error Handling *******

// Default ErrnoFun used for libzmq syscalls
func libZmqErrnoFun(errno os.Errno) os.Error {
  return os.NewError(C.GoString(C.zmq_strerror(C.int(errno))))
}

// {}
