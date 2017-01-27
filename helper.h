//
// Created by aclarembeau on 27/01/17.
//

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

struct addrWithPort {
    struct sockaddr *addr;
    size_t addr_len;
};

struct addrWithPort resolveAddrWithPort(char *host, size_t host_len, unsigned short port, int family_hint);
void freeAddrWithPort(struct addrWithPort str);

const char *sockaddrToString(struct sockaddr *addr, size_t addr_len);
int intptrToValue(void *intptr);
void *extractStatusPtr(struct syscall_res_subids idStruct);

#endif //GOLANG_MPTCP_API_HELPER_H
