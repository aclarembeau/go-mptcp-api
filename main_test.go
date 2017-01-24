package mptcp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
	"reflect"
	"time"
)

// V4 tests

var SOURCE_HOST string = "192.168.33.10"
var ANY_SOURCE_HOST string = ""
var DEST_HOST string = "130.104.230.45"

// V6 tests

/*var SOURCE_HOST string = "::"
var ANY_SOURCE_HOST string = "::"
var DEST_HOST string = "multipath-tcp.org"*/


// ---------------------------------------------------------------------------------------------------------------------

// Function used to establish a connection from sourceH:sourceP to destH:destP
// Fatal fail if the connection couldn't be established
func establishConnection(t *testing.T, sourceH string, sourceP string, destH string, destP string) *net.TCPConn {
	localAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(sourceH, sourceP))
	distantAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(destH, destP))
	conn, errCon := net.DialTCP("tcp", localAddr, distantAddr)
	if errCon != nil {
		t.Fatalf("Fatal error: unable to connect because %v", errCon)
	}

	return conn
}

// Logs a message depending on the result of the `passCondition` boolean variable.
// Shows "test OK" or a fancy error message indicating the excepted and the value we got
// in case of failure
func testExcepted(t *testing.T, passCondition bool, testName string, excepted string, got string) {
	if !passCondition {
		t.Errorf("[X] %s: excepted = %s, got = %s", testName, excepted, got)
	} else {
		t.Logf("[ ] %s: test OK!", testName)
	}
}

// Waits for 300 ms. This function is mainly used after opening or closing a subflow to wait
// for the operation to complete
func waitABit(){
	time.Sleep(300 * time.Millisecond)
}

// ---------------------------------------------------------------------------------------------------------------------

// Tests the function used to open the subflows
func TestOpenSub(t *testing.T) {
	// Establish the connection

	conn := establishConnection(t, SOURCE_HOST, "64100", DEST_HOST, "80")
	defer conn.Close()

	// Part 1: Opening 3 subflows and checking validity

	t.Log("Opening three valid subflows")
	fmt.Fprint(conn, "GET / HTTP/1.1\r\n")
	openedId1, openedError1 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "64101"), net.JoinHostPort(DEST_HOST, "80"))
	openedId2, openedError2 := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64102"), net.JoinHostPort(DEST_HOST, "80"))
	openedId3, openedError3 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "0"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	t.Log("Checking result")
	testExcepted(t, openedId1 == 2 && openedId2 == 3 && openedId3 == 4,
		"openedId value",
		"(2 3 4)",
		fmt.Sprintf("(%d %d %d)", openedId1, openedId2, openedId3))

	testExcepted(t, openedError1 == nil, "opening standard subflow",
		"success", fmt.Sprintf("error '%v'", openedError1))
	testExcepted(t, openedError2 == nil, "opening any source address subflow",
		"success", fmt.Sprintf("error '%v'", openedError2))
	testExcepted(t, openedError3 == nil, "opening any source port subflow",
		"success", fmt.Sprintf("error '%v'", openedError3))

	// Part 2 : Opening 3 erroneous subflows and checking the error

	t.Log("Opening three invalid subflows")
	_, err2 := OpenSub(nil, net.JoinHostPort(ANY_SOURCE_HOST, "64105"), net.JoinHostPort(DEST_HOST, "80"))
	_, err3 := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "20"), net.JoinHostPort(DEST_HOST, "80"))
	_, err4 := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64100"), net.JoinHostPort(DEST_HOST, "80"))

	t.Log("Checking result")
	testExcepted(t, err2 != nil, "opening nil-connection subflow", "failure", "success")
	testExcepted(t, err3 != nil, "opening source port < 1024 subflow", "failure", "success")
	testExcepted(t, err4 != nil, "opening already used subflow", "failure", "success")
}

// Tests the function used to get information about the subflow tuple
func TestGetSubTuple(t *testing.T) {
	// Establish the connection

	conn := establishConnection(t, SOURCE_HOST, "64200", DEST_HOST, "80")
	defer conn.Close()

	// Part 1: Inspect the main subflow

	t.Log("Inspecting subflow #1")
	fmt.Fprint(conn, "GET / HTTP/1.1\r\n")
	source1, dest1, inspectErr1 := GetSubTuple(conn, 1)

	testExcepted(t, inspectErr1 == nil, "inspection status", "success", fmt.Sprintf("error '%v'", inspectErr1))
	testExcepted(t,
		source1 == net.JoinHostPort(SOURCE_HOST, "64200") && dest1 == net.JoinHostPort(DEST_HOST, "80"),
		"inspection result",
		fmt.Sprintf("(%s %s)", net.JoinHostPort(SOURCE_HOST, "64200"), net.JoinHostPort(DEST_HOST, "80")),
		fmt.Sprintf("(%s %s)", source1, dest1))

	// Part 2: Open a new subflow and inspect it

	t.Log("Opening new subflow")
	_, openingError := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64201"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	testExcepted(t, openingError == nil, "opening new subflow", "success", fmt.Sprintf("error '%v'", openingError))

	t.Log("Inspecting subflow #2")
	source2, dest2, inspectErr2 := GetSubTuple(conn, 2)

	testExcepted(t, inspectErr2 == nil, "inspection status", "success", fmt.Sprintf("error '%v'", inspectErr2))
	testExcepted(t,
		source2 == net.JoinHostPort(SOURCE_HOST, "64201") && dest2 == net.JoinHostPort(DEST_HOST, "80"),
		"inspection result",
		fmt.Sprintf("(%s %s)", net.JoinHostPort(SOURCE_HOST, "64201"), net.JoinHostPort(DEST_HOST, "80")),
		fmt.Sprintf("(%s %s)", source2, dest2))

	// Part 3: Inspecting non-existing subflows

	t.Log("Inspecting invalid subflows")
	_, _, inspectErr3 := GetSubTuple(conn, 0)
	_, _, inspectErr4 := GetSubTuple(nil, 1)
	_, _, inspectErr5 := GetSubTuple(conn, 50)

	testExcepted(t, inspectErr3 != nil, "inspecting subflow #0", "failure", "success")
	testExcepted(t, inspectErr4 != nil, "inspecting nil connection", "failure", "success")
	testExcepted(t, inspectErr5 != nil, "inspecting subflow #50", "failure", "success")

}

// Tests the function used to remove the subflows
func TestCloseSub(t *testing.T) {
	// Establishing connection

	conn := establishConnection(t, SOURCE_HOST, "64300", DEST_HOST, "80")
	defer conn.Close()

	// Part 1 : Opening then closing subflows

	t.Log("Opening new subflow")
	fmt.Fprint(conn, "GET / HTTP/1.1\r\n")
	_, openingError := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64301"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	testExcepted(t, openingError == nil, "opening new subflow", "success", fmt.Sprintf("error '%v'", openingError))

	t.Log("Closing some subflows")
	closingErr1 := CloseSub(conn, 2, 0)
	testExcepted(t, closingErr1 == nil, "closing subflow #2", "success", fmt.Sprintf("error '%v'", closingErr1))
	closingErr2 := CloseSub(conn, 1, 0)
	testExcepted(t, closingErr2 == nil, "closing subflow #1", "success", fmt.Sprintf("error '%v'", closingErr2))

	// Part 2 : Closing non-existing subflow

	closingErr3 := CloseSub(conn, 42, 0)
	testExcepted(t, closingErr3 != nil, "closing subflow #45", "failure", "success")

}

func TestGetSubIDS(t *testing.T) {
	// Establishing connection

	conn := establishConnection(t, SOURCE_HOST, "64400", DEST_HOST, "80")
	defer conn.Close()

	// Part 1 : Opening four subflows and getting the list of subflows

	fmt.Fprint(conn, "GET / HTTP/1.1\r\n")

	t.Log("Opening 4 subflows")
	_, openingError1 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "64401"), net.JoinHostPort(DEST_HOST, "80"))
	_, openingError2 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "64402"), net.JoinHostPort(DEST_HOST, "80"))
	_, openingError3 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "64403"), net.JoinHostPort(DEST_HOST, "80"))
	_, openingError4 := OpenSub(conn, net.JoinHostPort(SOURCE_HOST, "64404"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	testExcepted(t, openingError1 == nil && openingError2 == nil && openingError3 == nil && openingError4 == nil,
		"opening subflows",
		"(nil, nil, nil, nil)",
		fmt.Sprintf("(%v %v %v %v)", openingError1, openingError2, openingError3, openingError4))

	t.Log("Listing subflows")
	list, getError := GetSubIDS(conn)

	testExcepted(t, getError == nil, "listing subflows", "success", fmt.Sprintf("error '%v'", getError))

	ids := []int{list[0][0], list[1][0], list[2][0], list[3][0], list[4][0]}
	prios := []int{list[0][1], list[1][1], list[2][1], list[3][1], list[4][1]}

	min := ids[0]
	for _, elem := range ids {
		if elem < min {min = elem; }
	}

	for index, _ := range ids {
		ids[index] -= min
	}

	testExcepted(t, reflect.DeepEqual([]int{4,3,2,1,0},ids), "checking subflow IDs - maxvalue", "4,3,2,1,0", fmt.Sprintf("%v", ids))
	testExcepted(t, reflect.DeepEqual([]int{0,0,0,0,0} ,prios), "checking subflow priorities", "0,0,0,0,0", fmt.Sprintf("%v", prios))
}

func TestSubsockopt(t *testing.T) {
	// Establishing the connection

	conn := establishConnection(t, SOURCE_HOST, "61500", DEST_HOST, "80")
	defer conn.Close()

	// Get command
	fmt.Fprint(conn, "GET / HTTP/1.1\r\n")
	t.Log("Opening new subflow")
	_, openingError := OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "61501"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	testExcepted(t, openingError == nil, "opening new subflow", "success", fmt.Sprintf("error '%v'", openingError))

	// Part 1 : Setting an option

	t.Log("Setting socket option")
	settingError := SetSubSockoptInt(conn, 2, syscall.SOL_IP, syscall.IP_TOS, 28)
	testExcepted(t, settingError == nil, "setting IP_TOS", "success", fmt.Sprintf("error '%v'", settingError))

	// Part 2 : Getting the value of the newly set option

	t.Log("Getting socket option")
	valMeasured, gettingError := GetSubSockoptInt(conn, 2, syscall.SOL_IP, syscall.IP_TOS)
	testExcepted(t, gettingError == nil, "getting IP_TOS", "success", fmt.Sprintf("error '%v'", gettingError))
	testExcepted(t, valMeasured == 28, "getting IP_TOS (value)", "28", fmt.Sprintf("%d", valMeasured))
}

// ---------------------------------------------------------------------------------------------------------------------

// Example about how to use the mptcp-api go binding. Works by trying to
// connect to multipath-tcp.org. Then, opens, lists, deletes and get information
// about the different subflows
func Example() {
	localAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(SOURCE_HOST, "64001"))
	distantAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(DEST_HOST, "80"))
	conn, _ := net.DialTCP("tcp", localAddr, distantAddr)
	defer conn.Close()

	// Get command
	fmt.Fprint(conn, "GET  / HTTP/1.1\r\n")

	// Opening 4 subflows
	OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64002"), net.JoinHostPort(DEST_HOST, "80"))
	OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64003"), net.JoinHostPort(DEST_HOST, "80"))
	OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64004"), net.JoinHostPort(DEST_HOST, "80"))
	OpenSub(conn, net.JoinHostPort(ANY_SOURCE_HOST, "64005"), net.JoinHostPort(DEST_HOST, "80"))
	waitABit()

	// Listing subflows
	fmt.Println(GetSubIDS(conn))

	// Closing one subflow
	fmt.Println(CloseSub(conn, 2, 0))
	waitABit()

	// Checking effect of the modification
	list, getError := GetSubIDS(conn)
	fmt.Println(getError)
	fmt.Println(list)

	// Now, getting even more details (source, port) tuples
	for _, element := range list {
		subflowId := element[0]
		fmt.Println(GetSubTuple(conn, subflowId))
	}

	// Setting an option
	SetSubSockoptInt(conn, 3, syscall.SOL_IP, syscall.IP_TOS, 28)

	// Getting the value
	fmt.Println(GetSubSockoptInt(conn, 3, syscall.SOL_IP, syscall.IP_TOS))

	// And checking that data are received
	var buf bytes.Buffer
	io.Copy(&buf, conn)
	fmt.Println("Len of data received: ", buf.Len())

}
