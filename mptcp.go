package mptcp


// #include <stdlib.h>
// #include "mptcplib.h"
// #include "helper.h"
import "C"
import (
	"net"
)


import (
	"os"
	"errors"
	"syscall"
	"fmt"
	"unsafe"
	"strconv"
)

// helpers -----------------------------------------

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

func errnoToError(functionName string, errnoValue int) error {
	if (errnoValue == 0) {
		return nil;
	} else if (errnoValue == C.EINVAL) {
		return fmt.Errorf("%s: Invalid subflow ID", functionName)
	} else if (errnoValue == C.EOPNOTSUPP) {
		return fmt.Errorf("%s: Operation not supported", functionName)
	} else if (errnoValue == C.ENOPROTOOPT) {
		return fmt.Errorf("%s: Option not valid at this level", functionName)
	} else {
		return fmt.Errorf("%s: Errno %d %s", functionName, errnoValue, C.GoString(C.strerror(C.int(errnoValue))))
	}
}

// exposed functions -------------------------------

type Subflow struct {
	local  string
	remote string
	id     int
	prio   int
}

func OpenSub(conn *net.TCPConn, flow *Subflow) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("OpenSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	localHost, localPort, errLocal := net.SplitHostPort(flow.local)
	remoteHost, remotePort, errRemote := net.SplitHostPort(flow.remote)

	if errLocal != nil {
		return fmt.Errorf("OpenSub: (decoding local) %v", errLocal)
	}
	if errRemote != nil {
		return fmt.Errorf("OpenSub: (decoding remote) %v", errRemote)
	}

	localPortInt, localPortError := strconv.Atoi(localPort)
	if localPortError != nil {
		return fmt.Errorf("OpenSub: (decoding local port) %v", localPortError)
	}
	remotePortInt, remotePortError := strconv.Atoi(remotePort)
	if remotePortError != nil {
		return fmt.Errorf("OpenSub: (decoding remote port) %v", remotePortError)
	}

	cLocalHost := C.CString(localHost)
	defer C.free(unsafe.Pointer(cLocalHost))
	cRemoteHost := C.CString(remoteHost)
	defer C.free(unsafe.Pointer(cRemoteHost))

	localStruct := C.resolveAddrWithPort(cLocalHost, C.size_t(len(localHost)), C.ushort(localPortInt), C.int(C.AF_UNSPEC))
	defer C.freeAddrWithPort(localStruct)
	if localStruct.addr == nil {
		return fmt.Errorf("OpenSub: (resolving local) Unable to resolve %s", localHost)
	}

	remoteStruct := C.resolveAddrWithPort(cRemoteHost, C.size_t(len(remoteHost)), C.ushort(remotePortInt), C.int(localStruct.addr.sa_family))
	defer C.freeAddrWithPort(remoteStruct)
	if remoteStruct.addr == nil {
		return fmt.Errorf("OpenSub: (resolving remote) Unable to resolve %s", remoteHost)
	}

	openedTuple := C.mptcplib_open_sub(C.int(fd), localStruct.addr, localStruct.addr_len, remoteStruct.addr, remoteStruct.addr_len, C.int(flow.prio))
	defer C.mptcplib_free_res_subtuple(&openedTuple)

	if (openedTuple.errnoValue != 0) {
		return errnoToError("OpenSub", int(openedTuple.errnoValue))
	}
	flow.id = int(openedTuple.id)

	return nil
}

func CloseSub(conn *net.TCPConn, subId int, how int) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("CloseSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	errnoValue := C.mptcplib_close_sub(C.int(fd), C.int(subId), C.int(how))
	return errnoToError("CloseSub", int(errnoValue))
}

func GetSubIDS(conn *net.TCPConn) ([]int, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return nil, fmt.Errorf("GetSubIDS: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	cStruct := C.mptcplib_get_sub_ids(C.int(fd))
	defer C.mptcplib_free_res_subids(&cStruct)

	if cStruct.errnoValue != 0 {
		return nil, errnoToError("GetSubIDS", int(cStruct.errnoValue))
	}

	nSubflows := int(cStruct.ids.sub_count)
	slice := (*[100]C.struct_mptcp_sub_status)(unsafe.Pointer(C.extractStatusPtr(cStruct)))[:nSubflows:nSubflows]

	idSlice := make([]int, len(slice))
	var i int
	for i = 0; i < nSubflows; i++ {
		idSlice[i] = int(slice[i].id)
	}

	return idSlice, nil
}

func GetSubTuple(conn *net.TCPConn, subId int) (Subflow, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return Subflow{}, fmt.Errorf("GetSub: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	cTuple := C.mptcplib_get_sub_tuple(C.int(fd), C.int(subId))
	defer C.mptcplib_free_res_subtuple(&cTuple)

	if (cTuple.errnoValue != 0) {
		return Subflow{}, errnoToError("GetSub", int(cTuple.errnoValue))
	}

	cLocalString := C.sockaddrToString(cTuple.local, cTuple.local_len)
	cRemoteString := C.sockaddrToString(cTuple.remote, cTuple.remote_len)
	defer C.free(unsafe.Pointer(cLocalString))
	defer C.free(unsafe.Pointer(cRemoteString))

	return Subflow{C.GoString(cLocalString), C.GoString(cRemoteString), int(cTuple.id), int(cTuple.low_prio), }, nil
}

func SetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int, optValue int) error {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return fmt.Errorf("SetSubSockoptInt: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	cInt := C.int(optValue)
	errnoValue := C.mptcplib_set_sub_sockopt(C.int(fd), C.int(subId), C.int(optLevel), C.int(optName), unsafe.Pointer(&cInt), C.size_t(unsafe.Sizeof(cInt)))

	return errnoToError("SetSubSockoptInt", int(errnoValue))
}

func GetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int) (int, error) {
	fd, fdFile, fdErr := getSockFd(conn)

	if fdErr != nil {
		return 0, fmt.Errorf("GetSubSockoptInt: (extracting fd) %v", fdErr)
	}

	defer syscall.SetNonblock(fd, true)
	defer fdFile.Close()

	// --

	dummyInt := C.int(0)
	cOption := C.mptcplib_get_sub_sockopt(C.int(fd), C.int(subId), C.int(optLevel), C.int(optName), C.size_t(unsafe.Sizeof(dummyInt)))
	defer C.mptcplib_free_res_sockopt(&cOption)

	if cOption.errnoValue != 0 {
		return 0, errnoToError("GetSubSockoptInt", int(cOption.errnoValue))
	}

	resultInt := C.intptrToValue(cOption.value)
	return int(resultInt), nil
}