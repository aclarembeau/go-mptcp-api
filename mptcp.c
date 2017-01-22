#include "mptcp.h"
#include <string.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <stdio.h>
#include <netinet/in.h>
#include <errno.h>

int setSubSockoptInt(int sockfd, int flowid, int level, int opt, int val){
	unsigned int optlen, sub_optlen;
	struct mptcp_sub_setsockopt sub_sso;

	optlen = sizeof(struct mptcp_sub_setsockopt);
	sub_optlen = sizeof(int);
	sub_sso.id = flowid;
	sub_sso.level = level;
	sub_sso.optname = opt;
	sub_sso.optlen = sub_optlen;
	sub_sso.optval = (char *) &val;

	int ret = setsockopt(sockfd, IPPROTO_TCP, MPTCP_SUB_SETSOCKOPT, &sub_sso, optlen);
	if(ret != 0) ret = errno;

	return ret;
}

struct getSubSockoptIntResult getSubSockoptInt(int sockfd, int flowid, int level, int opt){
	int error;

	int optlen = sizeof(struct mptcp_sub_getsockopt);
	struct mptcp_sub_getsockopt *get_info = malloc(optlen);

	int retval = 0;
	int sub_optlen = sizeof(int); 
	get_info->id = flowid;
	get_info->level = level;
	get_info->optname = opt;
	get_info->optlen = &sub_optlen;
	get_info->optval = (char *)&retval;

	error =  getsockopt(sockfd, IPPROTO_TCP, MPTCP_SUB_GETSOCKOPT, get_info, &optlen);
	if(error == -1) error = errno;

	struct getSubSockoptIntResult res = {error, retval};

	free(get_info);
	return res;
}

struct subflowInspectResult inspectSubflow(int sockfd, int flowid){
	// inspecting subflow
	unsigned int optlen = sizeof(struct sockaddr_in6) *2 + 40;

	struct mptcp_sub_tuple *sub_tuple = malloc(optlen);
	sub_tuple->id = flowid;

	int r= getsockopt(sockfd, IPPROTO_TCP, MPTCP_GET_SUB_TUPLE, sub_tuple,
			&optlen);


    if(r != 0){
        r = errno;
    }
	struct subflowInspectResult res = {r, NULL, NULL, 0, 0};
	if(r != 0){
	    free(sub_tuple);
        return res;
	}

	res.localHost = malloc(sizeof(char) * INET6_ADDRSTRLEN);    // Transmitted C-to-go #1: Must be freed
	res.distantHost = malloc(sizeof(char) * INET6_ADDRSTRLEN);  // Transmitted C-to-go #2: Must be freed

	void *sin = &sub_tuple->addrs[0];

	// extract string from source address
	struct sockaddr *stor = (struct sockaddr *)sin; // used to find the sa_family
	if(stor->sa_family == AF_INET){
		inet_ntop(AF_INET, &(((struct sockaddr_in *)sin)->sin_addr),res.localHost,INET6_ADDRSTRLEN);
		res.localPort = ntohs(((struct sockaddr_in *)sin)->sin_port);
		sin += sizeof(struct sockaddr_in);
	}
	else{
		inet_ntop(AF_INET6, &(((struct sockaddr_in6 *)sin)->sin6_addr),res.localHost,INET6_ADDRSTRLEN);
		res.localPort = ntohs(((struct sockaddr_in6 *)sin)->sin6_port);
		sin += sizeof(struct sockaddr_in6);
	}

	// extract string from distant address
	struct sockaddr *stor2 = (struct sockaddr *)sin; // used to find the sa_family
	if(stor2->sa_family == AF_INET){
		inet_ntop(AF_INET, &(((struct sockaddr_in *)sin)->sin_addr),res.distantHost,INET6_ADDRSTRLEN);
		res.distantPort = ntohs(((struct sockaddr_in *)sin)->sin_port);
		sin += sizeof(struct sockaddr_in);
	}
	else{
		inet_ntop(AF_INET6, &(((struct sockaddr_in6 *)sin)->sin6_addr),res.distantHost,INET6_ADDRSTRLEN);
		res.distantPort = ntohs(((struct sockaddr_in6 *)sin)->sin6_port);
		sin += sizeof(struct sockaddr_in6);
	}

	return res;
}

int closeSubflow(int sockfd, int flowid, int how){
	int error;

	int optlen = sizeof(struct mptcp_close_sub_id);
	struct mptcp_close_sub_id *close_info = malloc(optlen);

	close_info->id = flowid;
	//close_info->how = how;

	error =  getsockopt(sockfd, IPPROTO_TCP, MPTCP_CLOSE_SUB_ID, close_info, &optlen);
	if(error != 0) error = errno;

	free(close_info);
	return error;
}

struct openSubflowResult openSubflow(int sockfd, void *sourceaddr, int sourcelen, int sourceport, void *destaddr, int destlen, int destport){
	int error;

	int optlen = sizeof(struct mptcp_sub_tuple) + sourcelen + destlen;
	struct mptcp_sub_tuple *sub_tuple = malloc(optlen);

	sub_tuple->id = 0;
	//sub_tuple->prio = 0;

	void *addr = &sub_tuple->addrs[0];

    // copy the source address into the mptcp_sub_tuple structure
	memcpy(addr, sourceaddr, sourcelen);
	if(sourcelen == sizeof(struct sockaddr_in)){
		struct sockaddr_in *addr_v4 = (struct sockaddr_in *) addr;
		addr_v4->sin_port = htons(sourceport);
	}
	else if(sourcelen == sizeof(struct sockaddr_in6)){
		struct sockaddr_in6 *addr_v6 = (struct sockaddr_in6 *) addr;
		addr_v6->sin6_port = htons(sourceport);
	}

	addr += sourcelen;

    // copy the destination address into the mptcp_sub_tuple structure
	memcpy(addr, destaddr, destlen);
	if(destlen == sizeof(struct sockaddr_in)){
		struct sockaddr_in *addr_v4 = (struct sockaddr_in *) addr;
		addr_v4->sin_port = htons(destport);
	}
	else if(sourcelen == sizeof(struct sockaddr_in6)){
		struct sockaddr_in6 *addr_v6 = (struct sockaddr_in6 *) addr;
		addr_v6->sin6_port = htons(destport);
	}

    // do the system call
	error =  getsockopt(sockfd, IPPROTO_TCP, MPTCP_OPEN_SUB_TUPLE,
			sub_tuple, &optlen);
	if(error != 0) error = errno; // get the errno value

    // format and transmit to Go
    struct openSubflowResult res = {error, 0};
    if(error == 0) res.flowId = sub_tuple->id;

	free(sub_tuple);
	return res;
}



struct getSubflowsInfo getSubflows(int sockfd){
	int i;
	unsigned int optlen = SUBLIST_MIN_SIZE;
	struct mptcp_sub_ids *ids = malloc(optlen); // Transmitted C-to-go #3: Must be freed

	int r = EINVAL;
	while(r == EINVAL){
		// While we don't have enough space (EINVAL)
		free(ids);
		optlen += SUBLIST_INCREMENT;
		ids = malloc(optlen);
		r = getsockopt(sockfd, IPPROTO_TCP, MPTCP_GET_SUB_IDS, ids, &optlen);
		if(r != 0) r = errno;
	}


	if(r == 0)
	{
		struct getSubflowsInfo ret = {0, ids->sub_count, ids->sub_status, ids};
		return ret;
	}
	else{
		free(ids);
		struct getSubflowsInfo ret = {r, 0, NULL, NULL};
		return ret;
	}
}
