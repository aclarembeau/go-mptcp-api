/**
 * Library that simplifies the usage of the multipath-tcp socket API
 * (using less system calls, allowing seemlessly the usage of IPv4/IPV6, ...)
 *
 * By CLAREMBEAU Alexis
 * 01/24/2017
 */

#ifndef MPTCPLIB_H
#define MPTCPLIB_H

#include <errno.h>
#include <linux/tcp.h>
#include <stdint.h>
#include <wchar.h>

#define SUBLIST_MIN_SIZE 40 // Minimum allocated size
#define SUBLIST_INCREMENT 20 // Size increment for the "try & error" technique

// Structure for a MPTCP flow
struct mptcplib_flow {
    int id;                         // Subflow id
    int low_prio;                           // Subflow priority
    struct sockaddr *local, *remote;    // Local and remote endpoints
    size_t local_len, remote_len;          // Local and remote endpoints sizes
};

struct mptcplib_intarray {
    int count;
    int *values;
};

// Structure for the result of getsubtuple
struct mptcplib_getsubtuple_result {
    int errnoValue;
    struct mptcplib_flow flow ;
};

// Structure for the result of getsubids
struct mptcplib_getsubids_result {
    int errnoValue;
    struct mptcplib_intarray ids;
};

// Structure for the result of getsockopt
struct mptcplib_getsubsockopt_result {
    int errnoValue;
    void *value;
    size_t retsize;
};

/*
 * MPTCP manipulation functions
 */

// Creates a flow with given host and port and default priority
struct mptcplib_flow mptcplib_make_flow(char *source_host, unsigned short source_port, char *dest_host, unsigned short dest_port);

// Open a new subflow
int mptcplib_open_sub(int sockfd, struct mptcplib_flow *tuple);

// Close a specific subflow
int mptcplib_close_sub(int sockfd, int id, int how);

// Get all the subflow ids
struct mptcplib_getsubids_result mptcplib_get_sub_ids(int sockfd);

// Get information about a specific subflow
struct mptcplib_getsubtuple_result mptcplib_get_sub_tuple(int sockfd, int id);

// Set subflow socket options
int mptcplib_set_sub_sockopt(int sockfd, int id, int level, int opt, void *val, size_t size);

// Get subflow socket options
struct mptcplib_getsubsockopt_result mptcplib_get_sub_sockopt(int sockfd, int id, int level, int opt, size_t size);

/*
 * Memory freeing functions
 */
void mptcplib_free_intarray(struct mptcplib_intarray arr);
void mptcplib_free_flow(struct mptcplib_flow tuple);
void mptcplib_free_getsubtockopt_result(struct mptcplib_getsubsockopt_result sockopt);

#endif
