/**
 * Library that ports the multipath-tcp socket api from C to go
 * and that tries to simplify the usage using a more go idiomatic design.

 * By CLAREMBEAU Alexis
 * 01/28/2017
 */

package mptcp


// #include <stdlib.h>
// #include "mptcplib.h"
// #include "helper.h"
import "C"
import (
	"net"
	"os"
	"errors"
	"syscall"
	"fmt"
	"unsafe"
	"strconv"
)

// helpers -----------------------------------------

// gets the file descriptor that is represented by a socket
// (be careful, the underlying file is thus set blocking)
func getSockFd(conn *net.TCPConn) (C.int, *os.File, error) {
	if conn == nil {
		return 0, nil, errors.New("getSockFd: connection is nil")
	}

	fileInfo, fileError := conn.File()
	if fileError != nil {
		return 0, nil, fileError
	}
	sockfd := C.int(uint(fileInfo.Fd()))
	return sockfd, fileInfo, nil
}

// convert a errno value to a fancy error
// (using the convention defined in the socket api)
func errnoToError(functionName string, errnoValue int) error {
	if (errnoValue == 0) {
		return nil;
	} else if (errnoValue == C.EINVAL) {
		return fmt.Errorf("%s: Invalid subflow ID", functionName)
	} else {
		return fmt.Errorf("%s: Errno %d %s", functionName, errnoValue, C.GoString(C.strerror(C.int(errnoValue))))
	}
}

// exposed functions ---------------------------------------------------------------------------------------------------

// subflow structure, which defines a local and a remote endpoint and many informations such
// as the priority and the id
type Subflow struct {
	Local  string // local endpoint as form host:port
	Remote string // distant endpoint as form host:port
	Id     int    // subflow id
	Prio   int    // subflow priority (1 = low priority, 0 = normal)
}

// opens a new subflow from a TCP connection, assigns the id field of the flow parameter to the newly
// created flow and returns an appropriate error
func OpenSub(conn *net.TCPConn, flow *Subflow) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("OpenSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	// (1) Splitting host and port
	localHost, localPort, errLocal := net.SplitHostPort(flow.Local)
	remoteHost, remotePort, errRemote := net.SplitHostPort(flow.Remote)

	if errLocal != nil {
		return fmt.Errorf("OpenSub: (decoding local) %v", errLocal)
	}
	if errRemote != nil {
		return fmt.Errorf("OpenSub: (decoding remote) %v", errRemote)
	}

	// (2) Converting port to int
	localPortInt, localPortError := strconv.Atoi(localPort)
	if localPortError != nil {
		return fmt.Errorf("OpenSub: (decoding local port) %v", localPortError)
	}
	remotePortInt, remotePortError := strconv.Atoi(remotePort)
	if remotePortError != nil {
		return fmt.Errorf("OpenSub: (decoding remote port) %v", remotePortError)
	}

	// (3) Making C strings from go strings
	cLocalHost := C.CString(localHost)
	defer C.free(unsafe.Pointer(cLocalHost))
	cRemoteHost := C.CString(remoteHost)
	defer C.free(unsafe.Pointer(cRemoteHost))

	// (4) opening flow
	cFlow := C.mptcplib_make_flow(cLocalHost, C.ushort(localPortInt), cRemoteHost, C.ushort(remotePortInt))
	errnoValue := C.mptcplib_open_sub(fd, &cFlow)
	defer C.mptcplib_free_flow(cFlow)

	// (5) build the result
	if (errnoValue != 0) {
		return errnoToError("OpenSub", int(errnoValue))
	}
	flow.Id = int(cFlow.id)

	return nil
}

// close a subflow from the connection conn specified by its id and a parameter that
// indicates how the flow should be closed (by following the same convention as the shutdown
// system call)
func CloseSub(conn *net.TCPConn, subId int, how int) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("CloseSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	errnoValue := C.mptcplib_close_sub(fd, C.int(subId), C.int(how))
	return errnoToError("CloseSub", int(errnoValue))
}

// get the list of all subflow ids used in a given connection
func GetSubIDS(conn *net.TCPConn) ([]int, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return nil, fmt.Errorf("GetSubIDS: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	// (1) extract C structure
	cStruct := C.mptcplib_get_sub_ids(fd)
	defer C.mptcplib_free_getsubids_result(cStruct)

	if cStruct.errnoValue != 0 {
		return nil, errnoToError("GetSubIDS", int(cStruct.errnoValue))
	}

	// (2) make a go slice from sub_status
	nSubflows := int(cStruct.ids.sub_count)
	slice := (*[100]C.struct_mptcp_sub_status)(unsafe.Pointer(C.extractStatusPtr(cStruct)))[:nSubflows:nSubflows]

	// (3) only keep the id field
	idSlice := make([]int, len(slice))
	var i int
	for i = 0; i < nSubflows; i++ {
		idSlice[i] = int(slice[i].id)
	}

	// (4) return the result
	return idSlice, nil
}

// get the subflow with a given id in a connection
func GetSubTuple(conn *net.TCPConn, subId int) (*Subflow, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return nil, fmt.Errorf("GetSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	// (1) extract the C structure
	cStruct := C.mptcplib_get_sub_tuple(fd, C.int(subId))
	errnoValue := cStruct.errnoValue
	cFlow := cStruct.flow
	defer C.mptcplib_free_flow(cFlow)

	if (errnoValue != 0) {
		return nil, errnoToError("GetSub", int(errnoValue))
	}

	// (2) extract C strings
	cLocalString := C.sockaddrToString(cFlow.local, cFlow.local_len)
	cRemoteString := C.sockaddrToString(cFlow.remote, cFlow.remote_len)
	defer C.free(unsafe.Pointer(cLocalString))
	defer C.free(unsafe.Pointer(cRemoteString))

	// (3) convert C values and return subflow
	return &Subflow{C.GoString(cLocalString), C.GoString(cRemoteString), int(cFlow.id), int(cFlow.low_prio), }, nil
}

// set a subflow socket option (which should be a int value)
func SetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int, optValue int) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("SetSubSockoptInt: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	cInt := C.int(optValue)
	errnoValue := C.mptcplib_set_sub_sockopt(fd, C.int(subId), C.int(optLevel), C.int(optName), unsafe.Pointer(&cInt), C.size_t(unsafe.Sizeof(cInt)))

	return errnoToError("SetSubSockoptInt", int(errnoValue))
}

// get a subflow socket option (where the value should be an integer) 
func GetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int) (int, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return 0, fmt.Errorf("GetSubSockoptInt: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(int(fd), true)
	defer fdFile.Close()

	// -- upper part: see end of document -- 

	dummyInt := C.int(0)
	cOption := C.mptcplib_get_sub_sockopt(fd, C.int(subId), C.int(optLevel), C.int(optName), C.size_t(unsafe.Sizeof(dummyInt)))
	defer C.mptcplib_free_getsubtockopt_result(cOption)

	if cOption.errnoValue != 0 {
		return 0, errnoToError("GetSubSockoptInt", int(cOption.errnoValue))
	}

	resultInt := C.intptrToValue(cOption.value)
	return int(resultInt), nil
}

// ----------------------------------

/*
	On each function of the library, there is a common part which 
	basically extracts the underlying socket file descriptor from 
	a connection. 
	
	But, as extracting this value makes the socket blocking, it also 
	sets back the socket as non blocking using a defer call. 
 */