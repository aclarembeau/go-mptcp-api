# go-mptcp-api
Binding of the MPTCP socket api to go 

Allows applications to control the MPTCP stack from the applicative layer using the MPTCP socket API (https://tools.ietf.org/html/draft-hesmans-mptcp-socket-00). 

## Warnings

All the files and resources you can find on this repository are highly experimental. 
It is not encouraged to use it in production, mainly because of those elements: 

 * The native socket API itself is still experimental (not all features are implemented and maybe, its content can change in the future)
 * The signature of the Go methods is also susceptible to change in a close future (to be more consistent with the native API) 
 * The IP v6 functionalities hasn't been tested yet 
 * The absence of memory leaks also hasn't been thorously tested (Even if there shouldn't be any, I need to confirm their absence)
 * The library itself relies on the `.File().Fd()` methods, which involves many implications (and I need to study the extent of those **)

**: Go documentation indicates that 'The returned os.File's file descriptor is different from the connection's. Attempting to change properties of the original using this duplicate may or may not have the desired effect.' and 'File sets the underlying os.File to blocking mode and returns a copy. It is the caller's responsibility to close f when finished. Closing c does not affect f, and closing f does not affect c.'. 

The current code doesn't care about the difference between the original and the copied file decriptor. 
It also automatically set the socket in non-blocking mode after each API call. I must study carefully the impact of this choice. 

## How to use it

First, you need a MPTCP capable operating system with the native API. 
You can for instance use the excellent vagrant box available here: https://github.com/hoang-tranviet/mptcp-vagrant

Then, from the go process just grab the repository: 

    go get github.com/aclarembeau/go-mptcp-api
    
After that, the only step left is to install the library: 

    go install github.com/aclarembeau/go-mptcp-api
    
And you will be able to use the go binding in your code. 
Just import the appropriate package using: 

    import mptcp 
    
And set up a Go TCP Connection: 

    localAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(SOURCE_HOST, "64001"))
	  distantAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(DEST_HOST, "80"))
	  conn, _ := net.DialTCP("tcp", localAddr, distantAddr)
	  defer conn.Close()
    
Then, you can open, close or retrieve or set information about subflows using the functions: 

    func CloseSub(conn *net.TCPConn, subId int, how int) error
        Close a specific subflow. The parameter subId is used to indicate the
        subflow and the 'how' argument indicates how to close the subflow (this
        parameter is currently not used)

    func GetSubIDS(conn *net.TCPConn) ([][]int, error)
        Get all the subflows opened in the current TCP connection. Returns a
        list of tuples (id, priority). return = [[id subflow 1, priority subflow
        1], [id subflow 2, priority subflow 2], ... ]

    func GetSubSockoptInt(conn *net.TCPConn, subId int, level int, opt int) (int, error)
        Getsockopt for subflows. Takes as parameter the ID of the subflow, the
        level of the option and the option type

    func GetSubTuple(conn *net.TCPConn, subId int) (string, string, error)
        Inspect a subflow. Takes the ID of a subflow in the subId parameter.
        Returns two string representing local and distant endpoints using the
        host:port syntax.

    func OpenSub(conn *net.TCPConn, localEndpoint string, distantEndpoint string) (int, error)
        Opens a new subflow specified by its local and distant endpoint (in the
        host:port format)

    func SetSubSockoptInt(conn *net.TCPConn, subId int, level int, opt int, val int) error
        Setsockopt for subflows. Takes as parameter the ID of the subflow, the
        level of the option, the option type (integer) and the option value.

