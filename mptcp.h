#ifndef MPTCP_H
#define MPTCP_H

#include <errno.h>
#include <linux/tcp.h>

#define SUBLIST_MIN_SIZE 40 // Minimum allocated size
#define SUBLIST_INCREMENT 20 // Size increment for the "try & error" technique

// -- structures used to transmit information from C to GO ------------

/*
    Result of the inspect subflow function
*/
struct getSubTupleResult{
	int errnoValue;     // Value of errno (or 0 if the operation succeeded)
	char *localHost;    // Local host name
	char *distantHost;  // Distant host name
	int localPort;      // Local port
	int distantPort;    // Distant port
};

/*
    Result of the get subflow socket option function
*/
struct getSubSockoptIntResult{
	int errnoValue;     // Value of errno (or 0 if the operation succeeded)
	int result;         // getsockopt result
};

/*
    Result of the open subflow function
*/
struct openSubflowResult{
    int errnoValue; // Value of errno (or 0 if the operation succeeded)
    int flowId;     // ID of the flow created
};

/*
    Result of the get subflow info function
*/
struct getSubflowsInfo{
	int errnoValue;     // Value of errno (or 0 if the operation succeeded)
	int resultCount;    // Number of subflows
	void *resultPtr;    // Pointer to the list of subflows
	void *globalStructureTofree;    // Pointer to the malloc-ated structure (to free in go)
};

// -----------------------------------------

/**
    Get the list of all subflows (id of subflow, priority)
*/
struct getSubflowsInfo getSubIDS(int sockfd);

/**
    Opens a new subflow (specified with sockaddr pointers)
*/
struct openSubflowResult openSub(int sockfd, void *sourceaddr, int sourcelen, int sourceport, void *destaddr,
                                 int destlen, int destport);

/**
    Close a subflow specified by its id
*/
int closeSub(int sockfd, int flowid, int how);

/**
    Inspect a subflow (i.e getting source tuple: host and port, and destination tuple)
*/
struct getSubTupleResult getSubTuple(int sockfd, int flowid);

/**
    Getsockopt on a subflow
*/
struct getSubSockoptIntResult getSubSockoptInt(int sockfd, int flowid, int level, int opt);

/**
    Setsockopt on a subflow
*/
int setSubSockoptInt(int sockfd, int flowid, int level, int opt, int val);

#endif