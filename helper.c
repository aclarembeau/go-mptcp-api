#include "helper.h"


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

