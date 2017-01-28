/**
 * Test of the mptcp socket api go binding
 *
 * Designed to work on a machine with source addresses: 192.168.33.10 and 192.168.34.10
 *
 * By CLAREMBEAU Alexis
 * 01/28/2017
 */

package mptcp

import (
	"testing"
	"net"
	"fmt"
	"bufio"
	"syscall"
	"time"
)


// Function used to perform a test, with a test message, an excepted value and an error
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

	// making the net.TCPConn object
	localAddr, errLocal := net.ResolveTCPAddr("tcp", "192.168.33.10:64001")
	remoteAddr, errRemote := net.ResolveTCPAddr("tcp", "multipath-tcp.org:80")
	if errLocal != nil || errRemote != nil {
		t.Fatalf("Fatal error: unable to connect because %v %v", errLocal, errRemote)
	}
	conn, errCon := net.DialTCP("tcp", localAddr, remoteAddr)
	if errCon != nil {
		t.Fatalf("Fatal error: unable to connect because %v", errCon)
	}

	// sending a simple http request
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

	// (1) opening many subflows and performing some checks
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

	testExcepted(t, sub1.Id > 0, "subflow #1 id set by opensub", "> 0", fmt.Sprintf("%d", sub1.Id))
	testExcepted(t, sub2.Id > 0, "subflow #2 id set by opensub", "> 0", fmt.Sprintf("%d", sub2.Id))
	testExcepted(t, sub3.Id > 0, "subflow #3 id set by opensub", "> 0", fmt.Sprintf("%d", sub3.Id))

	// reading datas
	t.Log("Reading some data")
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Println(status, err)

	// (2) listing the opening subflows and checking consistency
	t.Log("Listing IDs and subflows")

	list1, errList1 := GetSubIDS(conn)
	testExcepted(t, errList1 == nil, "listing subflows", "succeeded", fmt.Sprintf("error: %v", errList1))
	testExcepted(t, len(list1) == 4, "counting list content", "== 4", fmt.Sprintf("%d", len(list1)))

	t.Log("Showing subflows")
	for _, id := range list1 {
		sub, errSub := GetSubTuple(conn, id)
		testExcepted(t, errSub == nil, "getting subflow tuple", "success", fmt.Sprintf("error %v", errSub))
		testExcepted(t, sub.Id >= 0, "getting subflow returned id", ">= 0", fmt.Sprintf("%d", sub.Id))
		testExcepted(t, sub.Prio == 1, "getting subflow returned prio", "== 1 (initialized value)", fmt.Sprintf("%d", sub.Prio))
	}
	_, errSubInv := GetSubTuple(conn, 100)
	testExcepted(t, errSubInv != nil, "Getting tuple of invalid subflow #100", "error", "success");

	// (3) removing some subflows and listing again to check the result
	t.Log("Removing subflows")

	errClose1 :=CloseSub(conn, list1[0], 0)
	testExcepted(t,errClose1 == nil,fmt.Sprintf("Closing subflow ID: %d", list1[0]) , "success", fmt.Sprintf("error %v", errClose1))
	errClose2 :=CloseSub(conn, list1[1], 0)
	testExcepted(t,errClose2 == nil,fmt.Sprintf("Closing subflow ID: %d", list1[1]) , "success", fmt.Sprintf("error %v", errClose2))
	errClose3 := CloseSub(conn, 100, 1)
	testExcepted(t, errClose3 != nil, "Closing invalid subflow #100", "error", "success")
	time.Sleep(100 * time.Millisecond)

	t.Log("Listing after")

	list2, errList2 := GetSubIDS(conn)
	testExcepted(t, errList2 == nil, "listing subflows", "succeeded", fmt.Sprintf("error: %v", errList2))

	for _, id := range list2 {
		testExcepted(t, id != list1[0] && id != list1[1], "Closed ID aren't present anymore", "success", fmt.Sprintf("%d still present", id))
	}

	// (4) Finally, setting and reading a socket option from a subflow
	t.Log("Testing socket options")

	errSet := SetSubSockoptInt(conn, list2[0], syscall.SOL_IP, syscall.IP_TOS, 28)
	testExcepted(t, errSet == nil, "Setting option to 28", "success", fmt.Sprintf("error %v", errSet))

	value, errGet := GetSubSockoptInt(conn, list2[0], syscall.SOL_IP, syscall.IP_TOS)
	testExcepted(t, errSet == nil, "Getting option", "success", fmt.Sprintf("error %v", errGet))
	testExcepted(t, value == 28, "Getting option value", "28 (set value)", fmt.Sprintf("%d", value))
}