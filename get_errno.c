#include <zmq.h>
#include <get_errno.h>

uint64_t get_errno() {
  // Default to zmq_errno which "does the right thing"
	return zmq_errno();
  // If this breaks, try plain errno, since zmq_errno is marked as experimental!
  // return errno;
}

