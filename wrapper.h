// wrapper.h

#ifndef WRAPPER_H
#define WRAPPER_H

#include <infiniband/verbs.h> // 包含内联函数的头文件

// 声明包装函数
int ibv_query_port_wrapper(struct ibv_context *context,
       uint8_t port_num, struct ibv_port_attr *port_attr);

uint16_t ibv_lid_wrapper(struct ibv_context *context,
       uint8_t port_num);

int ibv_modify_qp_wrapper(struct ibv_qp *qp,
         struct ibv_qp_attr *attr, int attr_mask);

int ibv_post_send_wrapper(struct ibv_qp *qp,
       struct ibv_send_wr *wr, struct ibv_send_wr **bad_wr, uint immData);

int ibv_get_imm_data(struct ibv_wc *wc);
#endif // WRAPPER_H
