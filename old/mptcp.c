#include "mptcp.h"
#include "../mptcplib.h"
#include <string.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <stdio.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>


int extractPort(struct sockaddr *addr){
    if (addr->sa_family == AF_INET) {
        return ntohs(((struct sockaddr_in *) addr)->sin_port);
    } else {
        return ntohs(((struct sockaddr_in6 *) addr)->sin6_port);
    }
}

struct sockaddr *injectPort(struct sockaddr *addr, int port){
    if (addr->sa_family == AF_INET) {
        ((struct sockaddr_in *) addr)->sin_port = htons(port);
    } else {
        ((struct sockaddr_in6 *) addr)->sin6_port = htons(port);
    }

    return addr;
}

// ---------------------------------------------------------------------

int setSubSockoptInt(int sockfd, int flowid, int level, int opt, int val) {
    return mptcplib_set_sub_sockopt(sockfd, flowid, level, opt, &val, sizeof(int));
}


struct getSubSockoptIntResult getSubSockoptInt(int sockfd, int flowid, int level, int opt) {
    struct syscall_res_sockopt res = mptcplib_get_sub_sockopt(sockfd, flowid, level, opt, sizeof(int));
    int intValue;
    if (res.errnoValue == 0) {
        intValue = *((int *) res.value);
    } else {
        intValue = 0;
    }

    struct getSubSockoptIntResult final_ret = {res.errnoValue, intValue};
    mptcplib_free_res_sockopt(&res);

    return final_ret;
}

struct getSubTupleResult getSubTuple(int sockfd, int flowid) {
    struct syscall_res_subtuple tuple = mptcplib_get_sub_tuple(sockfd, flowid);

    if (tuple.errnoValue == 0) {
        char *source = malloc(INET6_ADDRSTRLEN);
        char *dest = malloc(INET6_ADDRSTRLEN);


        int sourcePort, destPort = 0;
        if (tuple.local->sa_family == AF_INET) {
            inet_ntop(AF_INET,  &((struct sockaddr_in *) tuple.local)->sin_addr, source, INET6_ADDRSTRLEN);
            inet_ntop(AF_INET, &((struct sockaddr_in *) tuple.remote)->sin_addr, dest, INET6_ADDRSTRLEN);
        } else {
            inet_ntop(AF_INET,  &((struct sockaddr_in6 *) tuple.local)->sin6_addr, source, INET6_ADDRSTRLEN);
            inet_ntop(AF_INET, &((struct sockaddr_in6 *) tuple.remote)->sin6_addr, dest, INET6_ADDRSTRLEN);
        }

        struct getSubTupleResult res = {
                tuple.errnoValue,
                source, dest, extractPort(tuple.local), extractPort(tuple.remote)
        };
        mptcplib_free_res_subtuple(&tuple);

        return res;
    } else {

        struct getSubTupleResult res = {
                tuple.errnoValue,
                NULL, NULL, 0, 0
        };

        return res;
    }

}

struct openSubflowResult
openSub(int sockfd, void *sourceaddr, int sourcelen, int sourceport, void *destaddr, int destlen, int destport) {
    struct syscall_res_subtuple tuple = mptcplib_open_sub(
            sockfd,
            injectPort((struct sockaddr *) sourceaddr, sourceport), sourcelen,
            injectPort((struct sockaddr *) destaddr, destport), destlen,
            0);
    struct openSubflowResult res = {tuple.errnoValue, tuple.id};
    mptcplib_free_res_subtuple(&tuple);

    return res;
}


struct getSubflowsInfo getSubIDS(int sockfd) {
    struct syscall_res_subids res_struct = mptcplib_get_sub_ids(sockfd);


    struct getSubflowsInfo ret = {res_struct.errnoValue, res_struct.ids->sub_count, res_struct.ids->sub_status,
                                  res_struct.ids};
    if (res_struct.errnoValue != 0) {
        free(res_struct.ids);
    }

    return ret;
}