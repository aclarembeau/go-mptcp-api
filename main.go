// Experimental Go binding for the multipath-tcp api
// Based on the internet draft of B. Hesmans and O. Bonaventure
// from https://tools.ietf.org/html/draft-hesmans-mptcp-socket-00
package mptcp

// #include "mptcp.h"
// #include <stdlib.h>
import "C"

import "github.com/jbenet/go-sockaddr/net"
import "github.com/jbenet/go-sockaddr"
import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
	"errors"
	"strconv"
	"os"
)

// --- Helper unexported functions -------------------------------------------------------------------------------------

// Translate a errno value to a fancy error message
func getError(functName string, errno int) error {
	switch errno {
	case 0:
		return nil
	case C.EOPNOTSUPP:
		return fmt.Errorf("%s: <errno %d> (operation not supported)", functName, errno)
	case C.EINVAL:
		return fmt.Errorf("%s: <errno %d> (invalid subflow id)", functName, errno)
	case C.ENOPROTOOPT:
		return fmt.Errorf("%s: <errno %d> (this option is unknown at the level specified)", functName, errno)
	default:
		return fmt.Errorf("%s: <errno %d>", functName, errno)
	}
}

// Extracts the file descriptor from a TCPConn (and also sets the fd blocking)
func getSockFd(conn *net.TCPConn) (int, *os.File, error) {
	if conn == nil {
		return 0, nil, errors.New("getSockFd: connection is nil")
	}

	fileinfo, fileerror := conn.File()
	if fileerror != nil {
		return 0, nil, fileerror
	}
	sockfd := int(uint(fileinfo.Fd()))
	return sockfd, fileinfo, nil
}

// Resoves a host name to a sockaddr_any structure (and socklen value), if the protocol is specified
func resolveToSockaddrWithProto(host string, proto string) (*C.struct_sockaddr_any, int, error) {
	ip, err := net.ResolveIPAddr(proto, host)
	if err != nil {
		return nil, 0, err
	}
	sockaddrRaw := sockaddrnet.IPAddrToSockaddr(ip)
	sockaddrAny, socklen, _ := sockaddr.SockaddrToAny(sockaddrRaw)
	sockaddrC := sockaddr.AnyToCAny(sockaddrAny)

	return sockaddrC, int(socklen), err
}

// Resoves a host name to a sockaddr_any structure (and socklen value), with a constraint on the socklen
func resolveToSockaddrForced(host string, forced_len int) (*C.struct_sockaddr_any, int, error) {
	if(forced_len == 28){
		return resolveToSockaddrWithProto(host, "ip6")
	} else{
		return resolveToSockaddrWithProto(host, "ip4")
	}
}

// Resoves a host name to a sockaddr_any structure (and socklen value), with no constraints
func resolveToSockaddr(host string) (*C.struct_sockaddr_any, int, error) {
	return resolveToSockaddrWithProto(host, "ip")
}

// --- Library exported functions --------------------------------------------------------------------------------------

// Get all the subflows opened in the current TCP connection. Returns a list of tuples (id, priority).
// return = [[id subflow 1, priority subflow 1], [id subflow 2, priority subflow 2], ... ]
func GetSubIDS(conn *net.TCPConn) ([][]int, error) {
	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return nil, fmt.Errorf("getSubflows: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	// get the subflow list pointer
	subflowsInfo := C.getSubIDS(C.int(sockfd))
	nSubflows := int(subflowsInfo.resultCount)
	ptrSubflows := unsafe.Pointer(subflowsInfo.resultPtr)

	// handle the output
	err := getError("getSubflows", int(subflowsInfo.errnoValue))
	if err != nil {
		return nil, err
	}

	// convert everything to a go int[][]
	slice := (*[100]C.struct_mptcp_sub_status)(ptrSubflows)[:nSubflows:nSubflows]

	sliceGo := make([][]int, len(slice))
	var i int
	for i = 0; i < nSubflows; i++ {
		sliceGo[i] = []int{int(slice[i].id), -1}
	}

	// defer standard C free to old structure
	defer C.free(unsafe.Pointer(subflowsInfo.globalStructureTofree)) // Transmitted C-to-go #1: freed

	// and return the result
	return sliceGo, nil
}

// Inspect a subflow. Takes the ID of a subflow in the subId parameter. Returns two string representing local
// and distant endpoints using the host:port syntax.
func GetSubTuple(conn *net.TCPConn, subId int) (string, string, error) {
	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return "","", fmt.Errorf("inspectSubflow: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	// call the c function to inspect a subflow from the mptcp library
	subflowInspect := C.getSubTuple(C.int(sockfd), C.int(subId))

	err := getError("inspectSubflow", int(subflowInspect.errnoValue))
	if err != nil {
		return "","", err
	}

	localHostString := C.GoString(subflowInspect.localHost)
	distantHostString := C.GoString(subflowInspect.distantHost)
	defer C.free(unsafe.Pointer(subflowInspect.localHost))   // Transmitted C-to-go #2: freed
	defer C.free(unsafe.Pointer(subflowInspect.distantHost)) // Transmitted C-to-go #3: freed

	return net.JoinHostPort(localHostString, strconv.Itoa(int(subflowInspect.localPort))),
		net.JoinHostPort(distantHostString, strconv.Itoa(int(subflowInspect.distantPort))), nil
}

// Close a specific subflow. The parameter subId is used to indicate the subflow and the 'how' argument
// indicates how to close the subflow (this parameter is currently not used)
func CloseSub(conn *net.TCPConn, subId int, how int) error {
	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return fmt.Errorf("closeSubflow: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	// call the c function to close a subflow from the mptcp library
	cResultErrno := C.closeSub(C.int(sockfd), C.int(subId), C.int(how))

	// make fancy output depending on the error code
	return getError("closeSubflow", int(cResultErrno))
}

// Setsockopt for subflows. Takes as parameter the ID of the subflow, the level of the option, the option
// type (integer) and the option value.
func SetSubSockoptInt(conn *net.TCPConn, subId int, level int, opt int, val int) error {
	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return fmt.Errorf("getSubflowSockoptInt: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	// call the c function to add a subflow from the mptcp library
	cResultErrno := C.setSubSockoptInt(C.int(sockfd), C.int(subId), C.int(level), C.int(opt), C.int(val))

	return getError("setSubflowSockoptInt", int(cResultErrno))
}

// Getsockopt for subflows. Takes as parameter the ID of the subflow, the level of the option and the option
// type
func GetSubSockoptInt(conn *net.TCPConn, subId int, level int, opt int) (int, error) {
	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return 0, fmt.Errorf("getSubflowSockoptInt: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	// call the c function to add a subflow from the mptcp library
	cResult := C.getSubSockoptInt(C.int(sockfd), C.int(subId), C.int(level), C.int(opt))

	// make fancy output depending on the error code

	err := getError("getSubflowSockoptInt", int(cResult.errnoValue))
	if err == nil {
		return int(cResult.result), nil
	} else {
		return 0, err
	}
}

// Opens a new subflow specified by its local and distant endpoint (in the host:port format)
func OpenSub(conn *net.TCPConn, localEndpoint string, distantEndpoint string)(int, error){
	sourceHost, sourcePortString, errLocal := net.SplitHostPort(localEndpoint)
	destHost, destPortString, errDistant := net.SplitHostPort(distantEndpoint)

	if errLocal != nil {
		return 0, fmt.Errorf("openSubflow: (splitting sourcehost:sourceport) %v", errLocal)
	}
	if errDistant != nil {
		return 0, fmt.Errorf("openSubflow: (splitting desthost:destport) %v", errDistant)
	}

	sourcePort, sourcePortErr := strconv.Atoi(sourcePortString)
	destPort, destPortErr := strconv.Atoi(destPortString)

	if sourcePortErr != nil {
		return 0, fmt.Errorf("openSubflow: (getting sourceport) %v", sourcePortErr)
	}
	if destPortErr != nil {
		return 0, fmt.Errorf("openSubflow: (getting destport) %v", destPortErr)
	}

	// get the socket file descriptor
	sockfd, file, fdErr := getSockFd(conn)
	if fdErr != nil {
		return 0, fmt.Errorf("openSubflow: (extracting fd) %v", fdErr)
	}
	defer syscall.SetNonblock(sockfd, true) // Restablishing non-blocking property
	defer file.Close()

	sourceSockaddr, sourceSocklen, sourceError := resolveToSockaddr(sourceHost)
	if sourceError != nil {
		return 0, fmt.Errorf("openSubflow: (resolving source) %v", sourceError)
	}
	destSockaddr, destSocklen, destError := resolveToSockaddrForced(destHost, sourceSocklen)
	if destError != nil {
		return 0, fmt.Errorf("openSubflow: (resolving destination) %v", destError)
	}

	// call the c function to add a subflow from the mptcp library
	structResult := C.openSub(
		C.int(sockfd),
		unsafe.Pointer(sourceSockaddr), C.int(sourceSocklen), C.int(sourcePort),
		unsafe.Pointer(destSockaddr), C.int(destSocklen), C.int(destPort))

	// make fancy output depending on the error code
	return int(structResult.flowId), getError("openSubflow", int(structResult.errnoValue))
}
