#include "helper.h"

struct addrWithPort resolveAddrWithPort(char *host, size_t host_len, unsigned short port, int family_hint) {
    struct addrWithPort retVal = {NULL, 0};

    struct addrinfo hints;
    struct addrinfo *result;

    // information about the address to resolve
    memset(&hints, 0, sizeof(struct addrinfo));
    hints.ai_family = family_hint;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_flags = AI_PASSIVE;
    hints.ai_protocol = 0;
    hints.ai_canonname = NULL;
    hints.ai_addr = NULL;
    hints.ai_next = NULL;

    int s = getaddrinfo(host, NULL, &hints, &result);
    if (s != 0) {
        return retVal;
    }

    // allocate and copy the result
    retVal.addr = malloc(result->ai_addrlen);
    retVal.addr_len = result->ai_addrlen;
    memcpy(retVal.addr, result->ai_addr, result->ai_addrlen);

    // link with the port
    if (retVal.addr->sa_family == AF_INET) {
        ((struct sockaddr_in *) retVal.addr)->sin_port = htons(port);
    } else {
        ((struct sockaddr_in6 *) retVal.addr)->sin6_port = htons(port);
    }

    // and don't forget to free memory
    freeaddrinfo(result);

    return retVal;
}

void freeAddrWithPort(struct addrWithPort str) {
    if(str.addr != NULL)
        free(str.addr);
}

const char *sockaddrToString(struct sockaddr *addr, size_t addr_len) {
    // allocate string
    size_t bufLen = addr->sa_family == AF_INET ? INET_ADDRSTRLEN : INET6_ADDRSTRLEN;
    char *retString = malloc(sizeof(char) * bufLen);

    // find in_addr
    void *in_addr = NULL;
    if (addr->sa_family == AF_INET) {
        in_addr = (void *) &(((struct sockaddr_in *) addr)->sin_addr);
    } else {
        in_addr = (void *) &(((struct sockaddr_in6 *) addr)->sin6_addr);
    }

    // call ntop and return result
    const char *res = inet_ntop(addr->sa_family, in_addr, retString, bufLen);
    if(res == NULL){
        // in case of failure don't forget to free memory
        free(retString);
    }
    return res;
}

int intptrToValue(void *intptr) {
    int *ptrCasted = (int *) intptr;
    return *ptrCasted;
}


void *extractStatusPtr(struct syscall_res_subids idStruct) {
    return (void *) idStruct.ids->sub_status;
}

