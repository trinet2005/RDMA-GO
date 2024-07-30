// wrapper.c

#include <infiniband/verbs.h>
#include <stdio.h>
#define uint unsigned int
// package the ibv_query_port function
int ibv_query_port_wrapper(struct ibv_context *context,
       uint8_t port_num,struct ibv_port_attr *port_attr){
    return ibv_query_port(context,port_num,port_attr); // inlined function call
}

uint16_t ibv_lid_wrapper(struct ibv_context *context,
       uint8_t port_num){
    struct ibv_port_attr port_attr;
    ibv_query_port_wrapper(context,port_num,&port_attr);
    return port_attr.lid;
}

int ibv_modify_qp_wrapper(struct ibv_qp *qp,
       struct ibv_qp_attr *attr, int attr_mask){
    return ibv_modify_qp(qp,attr,attr_mask);
}

int ibv_post_send_wrapper(struct ibv_qp *qp,
       struct ibv_send_wr *wr, struct ibv_send_wr **bad_wr, uint immData){
    wr->imm_data = immData;
    return ibv_post_send(qp,wr,bad_wr);
}

int ibv_get_imm_data(struct ibv_wc *wc){
    return wc->imm_data;
}

