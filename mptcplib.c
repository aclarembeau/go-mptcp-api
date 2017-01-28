/**
 * Copied from:
 *  https://github.com/aclarembeau/simpler-mptcp-api/blob/master/mptcp.c
 */

#include "mptcplib.h"
#include <string.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <stdio.h>
#include <netinet/in.h>

struct syscall_res_subtuple mptcplib_open_sub(int sockfd,
                                              struct sockaddr *sourceaddr, size_t sourcelen,
                                              struct sockaddr *destaddr, size_t destlen,
                                              int prio) {
    // allocate the mptcp socket api structures
    int optlen = sizeof(struct mptcp_sub_tuple) + sourcelen + destlen;
    struct mptcp_sub_tuple *sub_tuple = malloc(optlen);

    sub_tuple->id = 0;
    //sub_tuple->prio = prio;

    void *addr = &sub_tuple->addrs[0];

    // do the system call
    memcpy(addr, sourceaddr, sourcelen);
    addr += sourcelen;
    memcpy(addr, destaddr, destlen);

    int error = getsockopt(sockfd, IPPROTO_TCP, MPTCP_OPEN_SUB_TUPLE, sub_tuple, &optlen);
    if (error != 0) error = errno; // get the errno value

    // format output and copy to structure

    struct sockaddr *copied_source = NULL;
    struct sockaddr *copied_dest = NULL;

    if(error == 0) {
        copied_source = malloc(sourcelen);
        copied_dest = malloc(destlen);
        memcpy(copied_source, sourceaddr, sourcelen);
        memcpy(copied_dest, destaddr, destlen);
    }

    struct syscall_res_subtuple res = {error, sub_tuple->id, prio, copied_source, copied_dest, sourcelen, destlen};

    // don't forget to free native structure
    free(sub_tuple);
    return res;
}


struct syscall_res_subids mptcplib_get_sub_ids(int sockfd) {
    // allocate the mptcp socket api structures
    int i;
    unsigned int optlen = SUBLIST_MIN_SIZE;
    struct mptcp_sub_ids *ids = malloc(optlen);

    int r = EINVAL;
    while (r == EINVAL) {
        // While we don't have enough space (EINVAL)
        free(ids);
        optlen += SUBLIST_INCREMENT;
        ids = malloc(optlen);
        r = getsockopt(sockfd, IPPROTO_TCP, MPTCP_GET_SUB_IDS, ids, &optlen);
        if (r != 0) r = errno;
    }

    // format the output
    if (r == 0) {
        struct syscall_res_subids ret = {0, ids};
        return ret;
    } else {
        free(ids);
        struct syscall_res_subids ret = {r, NULL};
        return ret;
    }
}


struct syscall_res_subtuple mptcplib_get_sub_tuple(int sockfd, int id) {
    // allocate the mptcp socket api structure
    unsigned int optlen = sizeof(struct sockaddr_in6) * 2 + 40;

    struct mptcp_sub_tuple *sub_tuple = malloc(optlen);
    sub_tuple->id = id;

    // do the system call
    int r = getsockopt(sockfd, IPPROTO_TCP, MPTCP_GET_SUB_TUPLE, sub_tuple, &optlen);

    if (r != 0) {
        r = errno;
    }
    int prio = -1;

    // format the output and copy local and remote addresses
    struct syscall_res_subtuple res = {r, id, prio, NULL, NULL, 0, 0};
    if (r != 0) {
        free(sub_tuple);
        return res;
    }

    void *sin = &sub_tuple->addrs[0];

    // local
    struct sockaddr *stor = (struct sockaddr *) sin; // used to find the sa_family
    if (stor->sa_family == AF_INET) {
        res.local_len = sizeof(struct sockaddr_in);
        res.local = malloc(res.local_len);
        memcpy(res.local, sin, res.local_len);
        sin += sizeof(struct sockaddr_in);
    } else {
        res.local_len = sizeof(struct sockaddr_in6);
        res.local = malloc(res.local_len);
        memcpy(res.local, sin, res.local_len);
        sin += sizeof(struct sockaddr_in6);
    }

    // remote
    struct sockaddr *stor2 = (struct sockaddr *) sin; // used to find the sa_family
    if (stor2->sa_family == AF_INET) {
        res.remote_len = sizeof(struct sockaddr_in);
        res.remote = malloc(res.remote_len);
        memcpy(res.remote, sin, res.remote_len);
        sin += sizeof(struct sockaddr_in);
    } else {
        res.remote_len = sizeof(struct sockaddr_in6);
        res.remote = malloc(res.remote_len);
        memcpy(res.remote, sin, res.remote_len);
        sin += sizeof(struct sockaddr_in6);
    }

    // don't forget to free internal structure
    free(sub_tuple);

    return res;
}

int mptcplib_close_sub(int sockfd, int id, int how) {
    // allocate the mptcp socket api structure
    struct mptcp_close_sub_id close_info;

    unsigned int optlen = sizeof(struct mptcp_close_sub_id);
    close_info.id = id;
    // close_info.how = how

    // do the system call
    int error = getsockopt(sockfd, IPPROTO_TCP, MPTCP_CLOSE_SUB_ID, &close_info, &optlen);
    if (error != 0) error = errno;

    return error;
}


int mptcplib_set_sub_sockopt(int sockfd, int id, int level, int opt, void *val, size_t size) {
    // allocate the mptcp socket api structure
    struct mptcp_sub_setsockopt sub_sso;
    memset(&sub_sso, 0, sizeof(sub_sso));

    unsigned int optlen = sizeof(struct mptcp_sub_setsockopt);
    sub_sso.id = id;
    sub_sso.level = level;
    sub_sso.optname = opt;
    sub_sso.optlen = size;
    sub_sso.optval = val;

    // do the system call
    int ret = setsockopt(sockfd, IPPROTO_TCP, MPTCP_SUB_SETSOCKOPT, &sub_sso, optlen);
    if (ret != 0) ret = errno;

    return ret;
}

struct syscall_res_sockopt mptcplib_get_sub_sockopt(int sockfd, int id, int level, int opt, size_t size) {
    // allocate the mptcp socket api structure
    struct mptcp_sub_getsockopt sub_sso;

    int retsize = size;
    char *retval = calloc(size, 1);
    unsigned int optlen = sizeof(struct mptcp_sub_setsockopt);
    sub_sso.id = id;
    sub_sso.level = level;
    sub_sso.optname = opt;
    sub_sso.optlen = &retsize;
    sub_sso.optval = retval;

    // get information
    int error = getsockopt(sockfd, IPPROTO_TCP, MPTCP_SUB_GETSOCKOPT, &sub_sso, &optlen);
    if (error == -1) error = errno;

    // format output
    struct syscall_res_sockopt res = {error, retval, retsize};
    return res;
}

/*
 * Memory freeing functions
 */
void mptcplib_free_res_subids(struct syscall_res_subids ids){
    free(ids.ids);
}
void mptcplib_free_res_subtuple(struct syscall_res_subtuple tuple){
    free(tuple.local);
    free(tuple.remote);
}
void mptcplib_free_res_sockopt(struct syscall_res_sockopt sockopt){
    free(sockopt.value);
}