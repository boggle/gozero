package zmq

// import "fmt"
import . "bytes"
import "io"
import "os"
import "unsafe"
import . "unsafe/coffer"

// #include <zmq.h>
// #include <stdlib.h>
import "C"


// ******** Global ZMQ Constants ********

const (
  ZmqPoll        = C.ZMQ_POLL
  ZmqP2P         = C.ZMQ_P2P // About to be renamed as ZMQ_PAIR
  ZmqPair        = C.ZMQ_P2P
  ZmqPub         = C.ZMQ_PUB
  ZmqSub         = C.ZMQ_SUB
  ZmqReq         = C.ZMQ_REQ
  ZmqRep         = C.ZMQ_REP
  ZmqXReq        = C.ZMQ_XREQ
  ZmqXRep        = C.ZMQ_XREP
  ZmqUpstream    = C.ZMQ_UPSTREAM
  ZmqDownstream  = C.ZMQ_DOWNSTREAM
  ZmqNoBlock     = C.ZMQ_NOBLOCK
  ZmqNoFlush     = C.ZMQ_NOFLUSH
  ZmqPollIn      = C.ZMQ_POLLIN
  ZmqPollOut     = C.ZMQ_POLLOUT
  ZmqPollErr     = C.ZMQ_POLLERR
  ZmqHWM         = C.ZMQ_HWM
  ZmqLWM         = C.ZMQ_LWM
  ZmqSwap        = C.ZMQ_SWAP
  ZmqAffinity    = C.ZMQ_AFFINITY
  ZmqIdentitiy   = C.ZMQ_IDENTITY
  ZmqRate        = C.ZMQ_RATE
  ZmqSubscribe   = C.ZMQ_SUBSCRIBE
  ZmqUnsubscribe = C.ZMQ_UNSUBSCRIBE
  ZmqRecoveryIVL = C.ZMQ_RECOVERY_IVL
  ZmqMCastLoop   = C.ZMQ_MCAST_LOOP
  ZmqSendBuf     = C.ZMQ_SNDBUF
  ZmqSndBuf      = C.ZMQ_SNDBUF
  ZmqRecvBuf     = C.ZMQ_RCVBUF
  ZmqRcvBuf      = C.ZMQ_RCVBUF
)


// ******** ZMQ Interfaces ********

type Provider interface {
  ErrKnow // implemented in utils.go

  NewContext(initArgs InitArgs) (Context, os.Error)
  NewMessage() Message

  // TODO Watch NewWatch()

  Version() (major int, minor int, pl int)

  Sleep(secs int)
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
// AppThreads = EnvGOMAXPROCS(), IoThreads = 2, Flags = 0 /*no polling! */
func DefaultInitArgs() InitArgs {
  return InitArgs{AppThreads: EnvGOMAXPROCS(), IoThreads: 2, Flags: 0}
}

// Context interface
//
// Contexts are always global thread-safe objects
type Context interface {
  io.Closer
  Provided

  NewSocket(socketType int) (Socket, os.Error)
  Terminate() os.Error
}

// Message interface
//
// Messages may only be used reliably with sockets from the same provider
type Message interface {
  io.Closer
  Provided

  GetData(buf *Buffer) (n int, err os.Error)
  SetData(buf *Buffer) (n int, err os.Error)

  // TODO Move(msg Message)
  // TODO Copy(msg Message)
  // TODO AliasData

  Size() int
}

// Socket interface
//
// Sockets are typically thread-bound
type Socket interface {
  io.Closer
  Provided

  Bind(address string) os.Error
  Connect(address string) os.Error

  Receive(msg Message, flags int) os.Error
  Send(msg Message, flags int) os.Error

  // TODO: Poll

  Flush() os.Error
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
// Don't forget to set EnvGOMAXPROCS appropriately when working with libzmq
//
// Contexts are finalized by the GC unless they are manually destructed
// by calling Terminate() beforehand.  Applications need to arrange
// that no socket is used or even closed after the owning context has
// been destructed.  This requires to have at least one running go routine
// with a live referene to the context.
func (p libZmqProvider) NewContext(args InitArgs) (Context, os.Error) {
  contextPtr := C.zmq_init(
    C.int(args.AppThreads),
    C.int(args.IoThreads),
    C.int(args.Flags))

  if isNull(contextPtr) {
    return nil, p.GetError()
  }

  lzmqContext := lzmqContext(contextPtr)
  return lzmqContext, nil
}

func (p *libZmqProvider) NewMessage() Message {
  msg := new(lzmqMessage)
  return msg
}

func (p *libZmqProvider) Sleep(secs int) {
  if secs < 0 {
    return
  }
  C.zmq_sleep(C.int(secs))
}

func (p *libZmqProvider) Version() (major int, minor int, pl int) {
  C.zmq_version((*C.int)(unsafe.Pointer(&major)), (*C.int)(unsafe.Pointer(&minor)), (*C.int)(unsafe.Pointer(&pl)))
  return
}

func (p lzmqContext) Provider() Provider { return LibZmqProvider() }

// Calls Terminate()
func (p lzmqContext) Close() os.Error {
  return p.Terminate()
}

// Calls zmq_term on underlying context pointer
//
// Only call once
func (p lzmqContext) Terminate() os.Error {
  ch := make(chan os.Error)
  ptr := unsafe.Pointer(p)
  if ptr != nil {
    // Needs to run in separate go routine to safely lock the OS Thread
    // and synchronize via channel to know when we're done
    Thunk(func() {
      ch <- p.Provider().OkIf(C.zmq_term(ptr) == 0)
    }).RunInOSThread()
    // Wait for completion
    return <-ch
  }
  return nil
}


// ******** lzmq: Messages ********

type lzmqMessageHolder interface {
  Message

  getLzmqMessage() *lzmqMessage
}

type lzmqMessage C.zmq_msg_t

func (p *lzmqMessage) empty() os.Error {
  return p.Provider().OkIf(C.zmq_msg_init(p.ptr()) == 0)
}

func (p *lzmqMessage) allocate(length int) os.Error {
  return p.Provider().OkIf(C.zmq_msg_init_size(p.ptr(), C.size_t(length)) == 0)
}

func (p *lzmqMessage) Provider() Provider { return LibZmqProvider() }

func (p *lzmqMessage) Size() int {
  // size_t always fits int, we do not allocate larger messages
  return int(C.zmq_msg_size(p.ptr()))
}

func (p *lzmqMessage) GetData(buf *Buffer) (n int, err os.Error) {
  n = p.Size()
  if n <= 0 {
    return 0, nil
  }
  start := p.data()
  var coffr *Coffer
  coffr, err = NewCoffer(start, n)
  if err != nil {
    return 0, err
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
    return 0, nil
  }
  err = p.allocate(n)
  if err != nil {
    return 0, err
  }

  start := p.data()
  var coffr *Coffer
  coffr, err = NewCoffer(start, n)
  if err != nil {
    return 0, err
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
  return p.Provider().OkIf(C.zmq_msg_close(p.ptr()) == 0)
}


// ******** lzmq: Sockets ********

// libzmq socket wrapper
type lzmqSocket uintptr

// Creates a new Socket with the given socketType
//
// Sockets only must be used from a fixed OSThread. This may be achieved
// by conveniently using Thunk.NewOSThread() or by calling runtime.LockOSThread()
func (p lzmqContext) NewSocket(socketType int) (Socket, os.Error) {
  ptr := unsafe.Pointer(C.zmq_socket(unsafe.Pointer(p), C.int(socketType)))
  if isNull(ptr) {
    return nil, p.Provider().GetError()
  }
  return lzmqSocket(ptr), nil
}

func (p lzmqSocket) Provider() Provider { return LibZmqProvider() }

// Bind server socket
func (p lzmqSocket) Bind(address string) os.Error {
  ptr := unsafe.Pointer(p)
  // apparantly freed by zmq
  c_addr := C.CString(address)
  return p.Provider().OkIf(C.zmq_bind(ptr, c_addr) == 0)
}

// Connect client socket
func (p lzmqSocket) Connect(address string) os.Error {
  ptr := unsafe.Pointer(p)
  // apparantly freed by zmq
  c_addr := C.CString(address)
  return p.Provider().OkIf(C.zmq_connect(ptr, c_addr) == 0)
}

func (p lzmqSocket) Receive(msg Message, flags int) os.Error {
  if msg == nil {
    return os.EINVAL
  }
  lzmqMsgHolder, castable := msg.(lzmqMessageHolder)
  if !castable {
    return os.EINVAL
  }
  lzmqMsg := lzmqMsgHolder.getLzmqMessage()
  err := lzmqMsg.empty()
  if err != nil {
    return err
  }
  ret := p.Provider().OkIf(C.zmq_recv(unsafe.Pointer(p), lzmqMsg.ptr(), C.int(flags)) == 0)
  // fmt.Println("recv", msg)
  return ret
}

func (p lzmqSocket) Send(msg Message, flags int) os.Error {
  if msg == nil {
    return os.EINVAL
  }
  lzmqMsgHolder, err := msg.(lzmqMessageHolder)
  if err == false {
    return os.EINVAL
  }
  lzmqMsg := lzmqMsgHolder.getLzmqMessage()
  ret := p.Provider().OkIf(C.zmq_send(unsafe.Pointer(p), lzmqMsg.ptr(), C.int(flags)) == 0)
  // fmt.Println("sent", msg)
  return ret
}

func (p lzmqSocket) Flush() os.Error {
  return p.Provider().OkIf(C.zmq_flush(unsafe.Pointer(p)) == 0)
}

// Closes this socket
//
// Expects the executing go routine to still be locked onto an OSThread.
// May be called only once
func (p lzmqSocket) Close() os.Error {
  return p.Provider().OkIf(C.zmq_close(unsafe.Pointer(p)) == 0)
}


func isNull(ptr unsafe.Pointer) bool {
	return uintptr(ptr) == uintptr(0)
}

// {}
