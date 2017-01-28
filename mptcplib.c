#include "mptcplib.h"
#include <string.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>

struct addrWithPort {
    struct sockaddr *addr;
    size_t addr_len;
};

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

// --------------

struct mptcplib_flow
mptcplib_make_flow(char *source_host, unsigned short source_port, char *dest_host, unsigned short dest_port) {
    struct addrWithPort local = resolveAddrWithPort(source_host, strlen(source_host),source_port, AF_UNSPEC );
    struct addrWithPort remote = resolveAddrWithPort(dest_host, strlen(dest_host),dest_port, local.addr->sa_family );

    struct mptcplib_flow res = {0, 0, local.addr, remote.addr, local.addr_len, remote.addr_len};
    return res;
}

int mptcplib_open_sub(int sockfd, struct mptcplib_flow *tuple) {
    // allocate the mptcp socket api structures
    int optlen = sizeof(struct mptcp_sub_tuple) + tuple->local_len + tuple->remote_len;
    struct mptcp_sub_tuple *sub_tuple = malloc(optlen);

    sub_tuple->id = 0;
    //sub_tuple->prio = prio;

    void *addr = &sub_tuple->addrs[0];

    memcpy(addr, tuple->local, tuple->local_len);
    addr += tuple->local_len;
    memcpy(addr, tuple->remote, tuple->remote_len);

    // do the system call
    int error = getsockopt(sockfd, IPPROTO_TCP, MPTCP_OPEN_SUB_TUPLE, sub_tuple, &optlen);
    if (error == 0) {
        tuple->id = sub_tuple->id;
    } else {
        error = errno;
    }

    // don't forget to free native structure
    free(sub_tuple);
    return error;
}


struct mptcplib_getsubids_result mptcplib_get_sub_ids(int sockfd) {
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
        // making the int array
        struct mptcplib_intarray arr = {ids->sub_count, malloc(sizeof(int) * ids->sub_count)};
        int i;
        for(i = 0 ; i < arr.count ; i++){
            arr.values[i] = ids->sub_status[i].id;
        }

        // building the result
        struct mptcplib_getsubids_result ret = {0, arr};
        free(ids);
        return ret;
    } else {
        free(ids);
        struct mptcplib_intarray arr = {0, NULL};
        struct mptcplib_getsubids_result ret = {r, arr};
        return ret;
    }
}


struct mptcplib_getsubtuple_result mptcplib_get_sub_tuple(int sockfd, int id) {
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
    struct mptcplib_flow res_flow = {id, prio, NULL, NULL, 0, 0};

    if (r != 0) {
        free(sub_tuple);
        struct mptcplib_getsubtuple_result res_final = {r, res_flow};
        return res_final;
    }

    void *sin = &sub_tuple->addrs[0];

    // local
    struct sockaddr *stor = (struct sockaddr *) sin; // used to find the sa_family
    if (stor->sa_family == AF_INET) {
        res_flow.local_len = sizeof(struct sockaddr_in);
        res_flow.local = malloc(res_flow.local_len);
        memcpy(res_flow.local, sin, res_flow.local_len);
        sin += sizeof(struct sockaddr_in);
    } else {
        res_flow.local_len = sizeof(struct sockaddr_in6);
        res_flow.local = malloc(res_flow.local_len);
        memcpy(res_flow.local, sin, res_flow.local_len);
        sin += sizeof(struct sockaddr_in6);
    }

    // remote
    struct sockaddr *stor2 = (struct sockaddr *) sin; // used to find the sa_family
    if (stor2->sa_family == AF_INET) {
        res_flow.remote_len = sizeof(struct sockaddr_in);
        res_flow.remote = malloc(res_flow.remote_len);
        memcpy(res_flow.remote, sin, res_flow.remote_len);
        sin += sizeof(struct sockaddr_in);
    } else {
        res_flow.remote_len = sizeof(struct sockaddr_in6);
        res_flow.remote = malloc(res_flow.remote_len);
        memcpy(res_flow.remote, sin, res_flow.remote_len);
        sin += sizeof(struct sockaddr_in6);
    }

    // don't forget to free internal structure
    free(sub_tuple);

    struct mptcplib_getsubtuple_result res_final = {r, res_flow};
    return res_final;
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

struct mptcplib_getsubsockopt_result mptcplib_get_sub_sockopt(int sockfd, int id, int level, int opt, size_t size) {
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
    struct mptcplib_getsubsockopt_result res = {error, retval, retsize};
    return res;
}

/*
 * Memory freeing functions
 */
void mptcplib_free_intarray(struct mptcplib_intarray arr){
    if(arr.values != NULL)
        free(arr.values);
}

void mptcplib_free_flow(struct mptcplib_flow tuple) {
    if(tuple.local != NULL)
        free(tuple.local);
    if(tuple.remote != NULL)
        free(tuple.remote);
}

void mptcplib_free_getsubtockopt_result(struct mptcplib_getsubsockopt_result sockopt) {
    if(sockopt.value != NULL)
        free(sockopt.value);
}
