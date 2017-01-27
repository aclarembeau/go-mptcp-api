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

/*
 * Structures used to gather data and error codes
 */

// List of subflow IDs and priorities
struct syscall_res_subids {
    int errnoValue;
    struct mptcp_sub_ids *ids;
};


// Subflow tuple
struct syscall_res_subtuple {
    int errnoValue;
    int id;                         // Subflow id
    int low_prio;                           // Subflow priority
    struct sockaddr *local, *remote;    // Local and remote endpoints
    size_t local_len, remote_len;          // Local and remote endpoints sizes
};


struct syscall_res_sockopt {
    int errnoValue;
    void *value;
    size_t retsize;
};

/*
 * Memory freeing functions
 */
void mptcplib_free_res_subids(struct syscall_res_subids *ids);
void mptcplib_free_res_subtuple(struct syscall_res_subtuple *tuple);
void mptcplib_free_res_sockopt(struct syscall_res_sockopt *sockopt);

/*
 * MPTCP manipulation functions
 */

// Open a new subflow
struct syscall_res_subtuple mptcplib_open_sub(int sockfd,
                                              struct sockaddr *sourceaddr, size_t sourcelen,
                                              struct sockaddr *destaddr, size_t destlen,
                                              int prio);

// Close a specific subflow
int mptcplib_close_sub(int sockfd, int id, int how);

// Get all the subflow ids
struct syscall_res_subids mptcplib_get_sub_ids(int sockfd);

// Get information about a specific subflow
struct syscall_res_subtuple mptcplib_get_sub_tuple(int sockfd, int id);

// Get subflow socket options
struct syscall_res_sockopt mptcplib_get_sub_sockopt(int sockfd, int id, int level, int opt, size_t size);

// Set subflow socket options
int mptcplib_set_sub_sockopt(int sockfd, int id, int level, int opt, void *val, size_t size);

#endif