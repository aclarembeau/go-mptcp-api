/**
 * Helper functions used to facilitate the transition of the logic
 * from C to Go
 *
 * By CLAREMBEAU Alexis
 * 01/28/2017
 */

#ifndef HELPER_H
#define HELPER_H

#include <stdlib.h>
#include <stdio.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <string.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <linux/tcp.h>
#include "mptcplib.h"

// Simple structure used to store a couple: sockaddr, sockaddrlen
struct addrWithPort {
    struct sockaddr *addr;
    size_t addr_len;
};

// Function used to resolve an address and to link it with a port
// ( there was functions in go to resolve addresses but not to use the ports )
struct addrWithPort resolveAddrWithPort(char *host, size_t host_len, unsigned short port, int family_hint);

// Free a addrWithPort structure
void freeAddrWithPort(struct addrWithPort str);

// Converts a sockaddr to a human readable string
const char *sockaddrToString(struct sockaddr *addr, size_t addr_len);

// Extracts the int value of a int* pointer represented by a void* variable
int intptrToValue(void *intptr);

// Extracts a mptcp_sub_status pointer from a syscall_res_subids structure
// (this can't be done in go because unsafe.Pointer doesn't work on null-sized arrays)
void *extractStatusPtr(struct syscall_res_subids idStruct);

#endif //GOLANG_MPTCP_API_HELPER_H
