# go-mptcp-api

The go multipath-tcp socket API is an (unofficial) binding of the MPTCP socket API defined in this document: https://tools.ietf.org/html/draft-hesmans-mptcp-socket-00. 

## How to use it? 

To use it, you need to have a multipath-tcp capable machine with a recent installation of the socket API. 
You can for instance use the vagrant box available at: https://github.com/hoang-tranviet/mptcp-vagrant

Then, after setting the GOPATH environment variable, you only have to do: 

```bash
	go get github.com/aclarembeau/go-mptcp-api
	go install github.com/aclarembeau/go-mptcp-api
```

After that, you would be able to use all the functions of the library. 

```
FUNCTIONS

func CloseSub(conn *net.TCPConn, subId int, how int) error
    close a subflow from the connection conn specified by its id and a
    parameter that indicates how the flow should be closed (by following the
    same convention as the shutdown system call)

func GetSubIDS(conn *net.TCPConn) ([]int, error)
    get the list of all subflow ids used in a given connection

func GetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int) (int, error)
    get a subflow socket option (where the value should be an integer)

func OpenSub(conn *net.TCPConn, flow *Subflow) error
    opens a new subflow from a TCP connection, assigns the id field of the
    flow parameter to the newly created flow and returns an appropriate
    error

func SetSubSockoptInt(conn *net.TCPConn, subId int, optLevel int, optName int, optValue int) error
    set a subflow socket option (which should be a int value)

func GetSubTuple(conn *net.TCPConn, subId int) (*Subflow, error)
    get the subflow with a given id in a connection

TYPES

type Subflow struct {
    Local  string // local endpoint as form host:port
    Remote string // distant endpoint as form host:port
    Id     int    // subflow id
    Prio   int    // subflow priority (1 = low priority, 0 = normal)
}

    subflow structure, which defines a local and a remote endpoint and many
    informations such as the priority and the id
```

## Warnings

Even if the library has been thorously tested (you can for instance do `go test github.com/aclarembeau/go-mptcp-api`), it is still in an experimental state and shouldn't be used in production.That's mainly because the multipath-tcp C socket API is itself experimental (all the functionalities aren't working yet, and that's why some tests are failing). But, the go API is also using some hacks in order to extract the file descriptor from a TCPConn object. In Go, to extract this information, I needed to turn the TCPConn to a File and then using the Fd method. This action has many implications and the official documentation specifies that: 
 
 > The returned os.File's file descriptor is different from the connection's. Attempting to change properties of the original using this duplicate may or may not have the desired effect.

The usage of the library should thus be restrained for testing purposes. 
If you see some strange behavior of you find a way to improve it, feel free to leave me a message (or do a pull request), it would always be a pleasure. 
