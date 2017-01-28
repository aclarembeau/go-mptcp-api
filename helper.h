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

// Converts a sockaddr to a human readable string
const char *sockaddrToString(struct sockaddr *addr, size_t addr_len);

// Extracts the int value of a int* pointer represented by a void* variable
int intptrToValue(void *intptr);

#endif //GOLANG_MPTCP_API_HELPER_H
