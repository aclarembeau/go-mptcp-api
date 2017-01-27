package mptcp

import (
	"testing"
	"net"
	"fmt"
	"bufio"
	"syscall"
)

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

func testExcepted(t *testing.T, passCondition bool, testName string, excepted string, got string) {
	if !passCondition {
		t.Errorf("[X] %s: excepted = %s, got = %s", testName, excepted, got)
	} else {
		t.Logf("[ ] %s: test OK!", testName)
	}
}


// Tests the function used to open the subflows
func TestOpenSub(t *testing.T) {
	t.Log("Making connection")

	conn := establishConnection(t, "::", "64001", "multipath-tcp.org", "80")
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

	t.Log("Opening subflows and checking")

	sub1 := Subflow{"192.168.33.10:64002", "multipath-tcp.org:80", -1, 1}
	sub2 := Subflow{"192.168.34.10:64003", "multipath-tcp.org:80", -1, 1}
	sub3 := Subflow{"[::]:64004", "multipath-tcp.org:80", -1, 1}
	sub4 := Subflow{"192.168.33.10:10", "multipath-tcp.org:80", -1, 1}
	sub5 := Subflow{"192.168.33.10:64002", "multipaath-tcp.org:80", -1, 1}

	errSub1 := OpenSub(conn, &sub1)
	errSub2 := OpenSub(conn, &sub2)
	errSub3 := OpenSub(conn, &sub3)
	errSub4 := OpenSub(conn, &sub4)
	errSub5 := OpenSub(conn, &sub5)

	testExcepted(t, errSub1 == nil, "opening subflow #1", "success", fmt.Sprintf("error: %v", errSub1))
	testExcepted(t, errSub2 == nil, "opening subflow #2", "success", fmt.Sprintf("error: %v", errSub2))
	testExcepted(t, errSub3 == nil, "opening subflow #3", "success", fmt.Sprintf("error: %v", errSub3))
	testExcepted(t, errSub4 != nil, "opening subflow #4: permission", "error", "success")
	testExcepted(t, errSub5 != nil, "opening subflow #5: resolve", "error", "success")

	testExcepted(t, sub1.id > 0, "subflow #1 id set by opensub", "> 0", fmt.Sprintf("%d", sub1.id))
	testExcepted(t, sub2.id > 0, "subflow #2 id set by opensub", "> 0", fmt.Sprintf("%d", sub2.id))
	testExcepted(t, sub3.id > 0, "subflow #3 id set by opensub", "> 0", fmt.Sprintf("%d", sub3.id))

	t.Log("Reading some data")

	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Println(status, err)

	t.Log("Listing IDs and subflows")

	list1, errList1 := GetSubIDS(conn)
	testExcepted(t, errList1 == nil, "listing subflows", "succeeded", fmt.Sprintf("error: %v", errList1))
	testExcepted(t, len(list1) == 4, "counting list content", "== 4", fmt.Sprintf("%d", len(list1)))

	t.Log("Showing subflows")

	for _, id := range list1 {
		sub, errSub := GetSubTuple(conn, id)
		testExcepted(t, errSub == nil, "getting subflow tuple", "success", fmt.Sprintf("error %v", errSub))
		testExcepted(t, sub.id >= 0, "getting subflow returned id", ">= 0", fmt.Sprintf("%d", sub.id))
		testExcepted(t, sub.prio == 1, "getting subflow returned prio", "== 1 (initialized value)", fmt.Sprintf("%d", sub.id))
	}
	_, errSubInv := GetSubTuple(conn, 100)
	testExcepted(t, errSubInv != nil, "Getting tuple of invalid subflow #100", "error", "success");

	t.Log("Removing subflows")

	errClose1 :=CloseSub(conn, list1[0], 0)
	testExcepted(t,errClose1 == nil,fmt.Sprintf("Closing subflow ID: %d", list1[0]) , "success", fmt.Sprintf("error %v", errClose1))
	errClose2 :=CloseSub(conn, list1[1], 0)
	testExcepted(t,errClose2 == nil,fmt.Sprintf("Closing subflow ID: %d", list1[1]) , "success", fmt.Sprintf("error %v", errClose2))
	errClose3 := CloseSub(conn, 100, 1)
	testExcepted(t, errClose3 != nil, "Closing invalid subflow #100", "error", "success")

	t.Log("Listing after")

	list2, errList2 := GetSubIDS(conn)
	testExcepted(t, errList2 == nil, "listing subflows", "succeeded", fmt.Sprintf("error: %v", errList2))

	for _, id := range list2 {
		testExcepted(t, id != list1[0] && id != list1[1], "Closed ID aren't present anymore", "success", fmt.Sprintf("%d still present", id))
	}

	t.Log("Testing socket options")

	errSet := SetSubSockoptInt(conn, list2[0], syscall.SOL_IP, syscall.IP_TOS, 28)
	testExcepted(t, errSet == nil, "Setting option to 28", "success", fmt.Sprintf("error %v", errSet))

	value, errGet := GetSubSockoptInt(conn, list2[0], syscall.SOL_IP, syscall.IP_TOS)
	testExcepted(t, errSet == nil, "Getting option", "success", fmt.Sprintf("error %v", errGet))
	testExcepted(t, value == 28, "Getting option value", "28 (set value)", fmt.Sprintf("%d", value))
}